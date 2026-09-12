package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
)

type RouteRequest struct {
	TaskType        string
	SafetyRisk      string
	RequireEvidence bool
	PersonaKey      string
	Policy          *domain.LLMRoutingPolicy
	Models          []domain.LLMModel
	IgnoreHealth    bool
}

type RouteDecision struct {
	PrimaryModelKey  string
	FallbackModelKey string
	Reason           string
}

type Router struct {
	Redis *redis.Client
}

func (r *Router) Decide(ctx context.Context, req RouteRequest) (RouteDecision, error) {
	if req.Policy == nil {
		return RouteDecision{}, ErrRoutingFailed
	}
	candidates := filterSelectableModels(req.Models, req.IgnoreHealth)
	if len(candidates) < 1 {
		return RouteDecision{}, ErrInsufficientModels
	}

	primary := strings.TrimSpace(req.Policy.DefaultModelKey)
	fallback := strings.TrimSpace(req.Policy.FallbackModelKey)

	preferred := preferredModelsForTask(req.Policy, req.TaskType)
	primaryCandidates := make([]string, 0, len(preferred))
	for _, key := range preferred {
		if containsModelKey(candidates, key) {
			primaryCandidates = append(primaryCandidates, key)
		}
	}
	if len(primaryCandidates) == 0 {
		if containsModelKey(candidates, primary) {
			primaryCandidates = []string{primary}
		} else {
			primaryCandidates = []string{candidates[0].ModelKey}
		}
	}

	primary = r.selectByLoadAndRoundRobin(ctx, primaryCandidates)
	if primary == "" {
		return RouteDecision{}, ErrInsufficientModels
	}

	if !containsModelKey(candidates, fallback) || fallback == primary {
		fallback = ""
		for _, m := range candidates {
			if m.ModelKey != primary {
				fallback = m.ModelKey
				break
			}
		}
	}
	if fallback == "" {
		return RouteDecision{}, ErrInsufficientModels
	}

	reason := fmt.Sprintf("task=%s;primary=%s;fallback=%s;policy=%s", req.TaskType, primary, fallback, req.Policy.Version)
	if req.RequireEvidence {
		reason += ";evidence_required=true"
	}
	if req.SafetyRisk != "" {
		reason += ";safety=" + req.SafetyRisk
	}
	if req.PersonaKey != "" {
		reason += ";persona=" + req.PersonaKey
	}
	return RouteDecision{PrimaryModelKey: primary, FallbackModelKey: fallback, Reason: reason}, nil
}

type modelLoad struct {
	Key  string
	Load int
}

func (r *Router) selectByLoadAndRoundRobin(ctx context.Context, keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	if len(keys) == 1 {
		if r.Redis != nil {
			_ = r.Redis.Set(ctx, roundRobinKey(), keys[0], 0)
		}
		return keys[0]
	}
	if r.Redis == nil {
		return keys[0]
	}

	loads := make([]modelLoad, 0, len(keys))
	minLoad := int(^uint(0) >> 1)
	for _, key := range keys {
		load, _ := r.Redis.Get(ctx, loadKey(key))
		n := atoi(load)
		loads = append(loads, modelLoad{Key: key, Load: n})
		if n < minLoad {
			minLoad = n
		}
	}

	tied := make([]string, 0, len(loads))
	for _, item := range loads {
		if item.Load == minLoad {
			tied = append(tied, item.Key)
		}
	}
	sort.Strings(tied)
	if len(tied) == 1 {
		r.TouchRoundRobin(ctx, tied[0])
		return tied[0]
	}

	last, _ := r.Redis.Get(ctx, roundRobinKey())
	pick := tied[0]
	for i, key := range tied {
		if key == last {
			pick = tied[(i+1)%len(tied)]
			break
		}
	}
	r.TouchRoundRobin(ctx, pick)
	return pick
}

func preferredModelsForTask(policy *domain.LLMRoutingPolicy, taskType string) []string {
	for _, rule := range policy.Rules {
		if rule.TaskType == taskType {
			return rule.PreferredModels
		}
	}
	return nil
}

func filterSelectableModels(models []domain.LLMModel, ignoreHealth bool) []domain.LLMModel {
	out := make([]domain.LLMModel, 0, len(models))
	for _, m := range models {
		if m.Status == domain.LLMModelActive || m.Status == domain.LLMModelDegraded {
			if !ignoreHealth && m.HealthStatus == domain.LLMHealthUnhealthy {
				continue
			}
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModelKey < out[j].ModelKey })
	return out
}

func containsModelKey(models []domain.LLMModel, key string) bool {
	for _, m := range models {
		if m.ModelKey == key {
			return true
		}
	}
	return false
}

func firstAvailable(preferred []string, models []domain.LLMModel) string {
	for _, key := range preferred {
		if containsModelKey(models, key) {
			return key
		}
	}
	return ""
}

func loadKey(modelKey string) string {
	return "llm:load:" + modelKey
}

func roundRobinKey() string {
	return "llm:router:round_robin_state"
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}

func (r *Router) IncLoad(ctx context.Context, modelKey string) {
	if r.Redis == nil {
		return
	}
	_ = r.Redis.Incr(ctx, loadKey(modelKey))
}

func (r *Router) DecLoad(ctx context.Context, modelKey string) {
	if r.Redis == nil {
		return
	}
	_ = r.Redis.Decr(ctx, loadKey(modelKey))
}

func (r *Router) TouchRoundRobin(ctx context.Context, modelKey string) {
	if r.Redis == nil {
		return
	}
	_ = r.Redis.Set(ctx, roundRobinKey(), modelKey, 0)
}
