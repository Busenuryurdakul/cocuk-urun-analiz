package llm

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ProviderKeyPrimary   = "primary"
	ProviderKeySecondary = "secondary"
	ModelKeyCareful      = "careful_analyst"
	ModelKeyResult       = "result_analyst"
)

func SeedPlatformDefaults(ctx context.Context, providers *repository.LLMProviderRepository, models *repository.LLMModelRepository, routing *repository.LLMRoutingPolicyRepository, personas *repository.LLMPersonaRepository, actorID primitive.ObjectID) error {
	primaryProvider := &domain.LLMProvider{
		ProviderKey: ProviderKeyPrimary,
		DisplayName: "Primary LLM Provider",
		Status:      domain.LLMProviderActive,
		SecretRef:   "env:LLM_PRIMARY_API_KEY",
		BaseURLRef:  "env:LLM_PRIMARY_BASE_URL",
	}
	if err := providers.Upsert(ctx, primaryProvider); err != nil {
		return err
	}
	secondaryProvider := &domain.LLMProvider{
		ProviderKey: ProviderKeySecondary,
		DisplayName: "Secondary LLM Provider",
		Status:      domain.LLMProviderActive,
		SecretRef:   "env:LLM_SECONDARY_API_KEY",
		BaseURLRef:  "env:LLM_SECONDARY_BASE_URL",
	}
	if err := providers.Upsert(ctx, secondaryProvider); err != nil {
		return err
	}

	careful := &domain.LLMModel{
		ProviderID:          primaryProvider.ID,
		ModelKey:            ModelKeyCareful,
		DisplayName:         "Careful Analyst",
		Status:              domain.LLMModelActive,
		ProviderModelName:   envOrDefault("LLM_PRIMARY_MODEL_NAME", "careful-analyst"),
		ContextWindowTokens: 128000,
		DefaultForPlatform:  true,
		HealthStatus:        domain.LLMHealthUnknown,
		Capabilities: domain.LLMModelCapabilities{
			Chat:               true,
			JSON:               true,
			SupportedTaskTypes: []string{"analysis", "review", "explanation"},
		},
	}
	if err := models.Upsert(ctx, careful); err != nil {
		return err
	}

	result := &domain.LLMModel{
		ProviderID:          secondaryProvider.ID,
		ModelKey:            ModelKeyResult,
		DisplayName:         "Result Analyst",
		Status:              domain.LLMModelActive,
		ProviderModelName:   envOrDefault("LLM_SECONDARY_MODEL_NAME", "result-analyst"),
		ContextWindowTokens: 128000,
		FallbackForPlatform: true,
		HealthStatus:        domain.LLMHealthUnknown,
		Capabilities: domain.LLMModelCapabilities{
			Chat:               true,
			JSON:               true,
			SupportedTaskTypes: []string{"analysis", "review", "decision_support"},
		},
	}
	if err := models.Upsert(ctx, result); err != nil {
		return err
	}

	if _, err := routing.FindPublished(ctx, nil, domain.LLMPlatformDefaultRoutingVersion); err != nil {
		if err != repository.ErrNotFound {
			return err
		}
		now := time.Now().UTC()
		policy := &domain.LLMRoutingPolicy{
			Version:          domain.LLMPlatformDefaultRoutingVersion,
			Status:           domain.LLMConfigPublished,
			DefaultModelKey:  ModelKeyCareful,
			FallbackModelKey: ModelKeyResult,
			Rules: []domain.LLMRoutingRule{
				{TaskType: "analysis", PreferredModels: []string{ModelKeyCareful, ModelKeyResult}},
				{TaskType: "review", PreferredModels: []string{ModelKeyResult, ModelKeyCareful}},
				{TaskType: "deep_analysis", PreferredModels: []string{ModelKeyResult, ModelKeyCareful}},
			},
			Reason:      "platform bootstrap routing policy",
			CreatedBy:   actorID,
			PublishedAt: &now,
		}
		if err := routing.Insert(ctx, policy); err != nil {
			return err
		}
	}

	for _, spec := range []struct {
		key, name, instruction string
	}{
		{domain.PersonaCarefulAnalyst, "Careful Analyst", "You are a careful, evidence-focused analyst for Miyuna children's product decision-support. Explain cautiously, stay compliance-aware, and never claim certainty or definitive safety/regulatory verdicts without verified evidence."},
		{domain.PersonaResultAnalyst, "Result Analyst", "You are a concise decision-support analyst for Miyuna. Focus on actionable outcomes and risk signals. Do not assert regulatory compliance, certification, or guaranteed safety."},
	} {
		if _, err := personas.FindPublished(ctx, nil, spec.key, domain.LLMPlatformDefaultPersonaVersion); err != nil {
			if err != repository.ErrNotFound {
				return err
			}
			now := time.Now().UTC()
			persona := &domain.LLMPersona{
				PersonaKey:        spec.key,
				Version:           domain.LLMPlatformDefaultPersonaVersion,
				Status:            domain.LLMConfigPublished,
				DisplayName:       spec.name,
				SystemInstruction: spec.instruction,
				Reason:            "platform bootstrap persona",
				CreatedBy:         actorID,
				PublishedAt:       &now,
			}
			if err := personas.Insert(ctx, persona); err != nil {
				return err
			}
		}
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func ResolveSecretRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "env:") {
		return strings.TrimSpace(os.Getenv(strings.TrimPrefix(ref, "env:")))
	}
	return ""
}

func ResolveBaseURLRef(ref string) string {
	return ResolveSecretRef(ref)
}
