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
	TaskType         string
	SafetyRisk       string
	RequireEvidence  bool
	PersonaKey       string
	OrgDefaultKey    string
	OrgFallbackKey   string
	Policy           *domain.LLMRoutingPolicy
	Models           []domain.LLMModel
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
	candidates := filterSelectableModels(req.Models)
	if len(candidates) < 1 {
		return RouteDecision{}, ErrInsufficientModels
	}

	primary := strings.TrimSpace(req.OrgDefaultKey)
	if primary == "" {
		primary = strings.TrimSpace(req.Policy.DefaultModelKey)
	}
	fallback := strings.TrimSpace(req.OrgFallbackKey)
	if fallback == "" {
		fallback = strings.TrimSpace(req.Policy.FallbackModelKey)
	}

	preferred := preferredModelsForTask(req.Policy, req.TaskType)
	if len(preferred) > 0 {
		if pick := firstAvailable(preferred, candidates); pick != "" {
			primary = pick
		}
	}

	if !containsModelKey(candidates, primary) {
		primary = candidates[0].ModelKey
	}
	if fallback == "" || fallback == primary {
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

	// Tie-break using least-loaded then round-robin when multiple equally preferred.
	if r.Redis != nil && len(preferred) > 1 {
		loads := make([]modelLoad, 0, len(preferred))
		for _, key := range preferred {
			if !containsModelKey(candidates, key) {
				continue
			}
			load, _ := r.Redis.Get(ctx, loadKey(key))
			loads = append(loads, modelLoad{Key: key, Load: atoi(load)})
		}
		if len(loads) > 0 {
			sort.Slice(loads, func(i, j int) bool {
				if loads[i].Load == loads[j].Load {
					return loads[i].Key < loads[j].Key
				}
				return loads[i].Load < loads[j].Load
			})
			primary = loads[0].Key
		}
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

func preferredModelsForTask(policy *domain.LLMRoutingPolicy, taskType string) []string {
	for _, rule := range policy.Rules {
		if rule.TaskType == taskType {
			return rule.PreferredModels
		}
	}
	return nil
}

func filterSelectableModels(models []domain.LLMModel) []domain.LLMModel {
	out := make([]domain.LLMModel, 0, len(models))
	for _, m := range models {
		if m.Status == domain.LLMModelActive || m.Status == domain.LLMModelDegraded {
			if m.HealthStatus != domain.LLMHealthUnhealthy {
				out = append(out, m)
			}
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
