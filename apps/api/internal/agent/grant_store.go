package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const grantKeyPrefix = "agent:grant:"

var consumeGrantScript = `
local v = redis.call("GET", KEYS[1])
if not v then return 0 end
redis.call("DEL", KEYS[1])
return v
`

type GrantRecord struct {
	GrantIDHash     string `json:"grantIdHash"`
	OrganizationID  string `json:"organizationId"`
	UserID          string `json:"userId"`
	AnalysisRunID   string `json:"analysisRunId"`
	ToolExecutionID string `json:"toolExecutionId"`
	ToolName        string `json:"toolName"`
	ToolVersion     string `json:"toolVersion"`
	RegistryVersion string `json:"registryVersion"`
	InputHash       string `json:"inputHash"`
	TraceID         string `json:"traceId"`
	IssuedAt        int64  `json:"issuedAt"`
	ExpiresAt       int64  `json:"expiresAt"`
}

type GrantBinding struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	UserID         primitive.ObjectID
	ToolName       string
	ToolVersion    string
	InputHash      string
	TraceID        string
}

type GrantStore struct {
	Redis    *redis.Client
	Security *repository.SecurityEventRepository
	TTL      time.Duration
}

func HashGrantToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (g *GrantStore) Issue(ctx context.Context, grant *AuthorizationGrant, registryVersion, traceID string) error {
	if g == nil || g.Redis == nil {
		return ErrGrantStoreUnavailable
	}
	ttl := g.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now().UTC()
	grant.ExpiresAt = now.Add(ttl)
	record := GrantRecord{
		GrantIDHash:     HashGrantToken(grant.Nonce),
		OrganizationID:  grant.OrganizationID.Hex(),
		UserID:          grant.UserID.Hex(),
		AnalysisRunID:   grant.AnalysisRunID.Hex(),
		ToolExecutionID: grant.ToolExecutionID.Hex(),
		ToolName:        grant.ToolName,
		ToolVersion:     grant.ToolVersion,
		RegistryVersion: registryVersion,
		InputHash:       grant.InputHash,
		TraceID:         traceID,
		IssuedAt:        now.Unix(),
		ExpiresAt:       grant.ExpiresAt.Unix(),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	key := grantKeyPrefix + record.GrantIDHash
	return g.Redis.Set(ctx, key, string(payload), ttl)
}

func (g *GrantStore) Consume(ctx context.Context, nonce string, binding GrantBinding) (*GrantRecord, error) {
	if g == nil || g.Redis == nil {
		return nil, ErrGrantStoreUnavailable
	}
	hash := HashGrantToken(nonce)
	key := grantKeyPrefix + hash
	raw, err := g.Redis.Eval(ctx, consumeGrantScript, []string{key})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGrantStoreUnavailable, err)
	}
	if raw == nil || raw == int64(0) || raw == "" {
		g.recordReplay(ctx, binding, "grant_missing")
		return nil, ErrGrantReplay
	}
	var record GrantRecord
	switch v := raw.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &record); err != nil {
			return nil, err
		}
	default:
		return nil, ErrGrantReplay
	}
	if time.Now().UTC().Unix() > record.ExpiresAt {
		g.recordReplay(ctx, binding, "grant_expired")
		return nil, ErrGrantExpired
	}
	if record.OrganizationID != binding.OrganizationID.Hex() ||
		record.AnalysisRunID != binding.AnalysisRunID.Hex() ||
		record.ToolName != binding.ToolName ||
		record.InputHash != binding.InputHash {
		g.recordReplay(ctx, binding, "grant_binding_mismatch")
		return nil, ErrAuthorizationDenied
	}
	return &record, nil
}

func (g *GrantStore) recordReplay(ctx context.Context, binding GrantBinding, reason string) {
	if g.Security == nil {
		return
	}
	uid := binding.UserID
	oid := binding.OrganizationID
	_ = g.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventUnauthorizedTool,
		Severity:       domain.SeverityWarning,
		Details: map[string]string{
			"reason":        reason,
			"tool":          binding.ToolName,
			"analysisRunId": binding.AnalysisRunID.Hex(),
		},
	})
}
