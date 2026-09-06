package graph

import (
	"errors"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	llmsvc "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/llm"
)

func mapPhase6Error(err error) error {
	switch {
	case errors.Is(err, llmsvc.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, llmsvc.ErrInvalidInput):
		return gqlError("INVALID_INPUT", err)
	case errors.Is(err, llmsvc.ErrPublishValidation), errors.Is(err, llmsvc.ErrDraftNotValidated):
		return gqlError("LLM_PUBLISH_VALIDATION_FAILED", err)
	case errors.Is(err, llmsvc.ErrInvalidPolicyVersion):
		return gqlError("LLM_ROUTING_POLICY_INVALID", err)
	case errors.Is(err, llmsvc.ErrComplianceBlocked), errors.Is(err, llmsvc.ErrPromptInjection):
		return gqlError("COMPLIANCE_BLOCKED", err)
	case errors.Is(err, llmsvc.ErrInsufficientModels):
		return gqlError("LLM_MODEL_DISABLED", err)
	default:
		return mapPhase5Error(err)
	}
}

func toModelLLMProviders(items []domain.LLMProvider) []*model.LLMProvider {
	out := make([]*model.LLMProvider, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, &model.LLMProvider{
			ID:          item.ID.Hex(),
			ProviderKey: item.ProviderKey,
			DisplayName: item.DisplayName,
			Status:      string(item.Status),
		})
	}
	return out
}

func toModelLLMModels(items []domain.LLMModel) []*model.LLMModel {
	out := make([]*model.LLMModel, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, &model.LLMModel{
			ID:                  item.ID.Hex(),
			ModelKey:            item.ModelKey,
			DisplayName:         item.DisplayName,
			Status:              string(item.Status),
			HealthStatus:        string(item.HealthStatus),
			ContextWindowTokens: item.ContextWindowTokens,
			DefaultForPlatform:  item.DefaultForPlatform,
			FallbackForPlatform: item.FallbackForPlatform,
			SupportedTaskTypes:  item.Capabilities.SupportedTaskTypes,
		})
	}
	return out
}

func toModelLLMRoutingPolicies(items []domain.LLMRoutingPolicy) []*model.LLMRoutingPolicy {
	out := make([]*model.LLMRoutingPolicy, 0, len(items))
	for i := range items {
		item := items[i]
		m := &model.LLMRoutingPolicy{
			ID:               item.ID.Hex(),
			Version:          item.Version,
			Status:           string(item.Status),
			DefaultModelKey:  item.DefaultModelKey,
			FallbackModelKey: item.FallbackModelKey,
		}
		if item.PublishedAt != nil {
			v := item.PublishedAt.UTC().Format(time.RFC3339)
			m.PublishedAt = &v
		}
		out = append(out, m)
	}
	return out
}

func toModelLLMPersonas(items []domain.LLMPersona) []*model.LLMPersona {
	out := make([]*model.LLMPersona, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, &model.LLMPersona{
			ID:          item.ID.Hex(),
			PersonaKey:  item.PersonaKey,
			Version:     item.Version,
			Status:      string(item.Status),
			DisplayName: item.DisplayName,
		})
	}
	return out
}

func toModelLLMConfiguration(snap *domain.ConfigSnapshot) *model.LLMConfiguration {
	out := &model.LLMConfiguration{
		ID:                   snap.ID.Hex(),
		RoutingPolicyVersion: snap.LLMRoutingPolicyVersion,
		PersonaKey:           snap.LLMPersonaKey,
		PersonaVersion:       snap.LLMPersonaVersion,
		DefaultModelKey:      snap.DefaultModelKey,
		FallbackModelKey:     snap.FallbackModelKey,
		PublishedAt:          snap.PublishedAt.UTC().Format(time.RFC3339),
		Reason:               snap.Reason,
	}
	if snap.OrganizationID != nil {
		v := snap.OrganizationID.Hex()
		out.OrganizationID = &v
	}
	return out
}

func toModelLLMConfigurationDraft(draft *domain.LLMConfigurationDraft) *model.LLMConfigurationDraft {
	orgID := ""
	if draft.OrganizationID != nil {
		orgID = draft.OrganizationID.Hex()
	}
	return &model.LLMConfigurationDraft{
		ID:                   draft.ID.Hex(),
		OrganizationID:       orgID,
		Status:               string(draft.Status),
		RoutingPolicyVersion: draft.RoutingPolicyVersion,
		PersonaKey:           draft.PersonaKey,
		PersonaVersion:       draft.PersonaVersion,
		DefaultModelKey:      draft.DefaultModelKey,
		FallbackModelKey:     draft.FallbackModelKey,
		ValidationErrors:     draft.ValidationErrors,
		UpdatedAt:            draft.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toModelLLMOrgSettings(settings *domain.LLMOrgSettings) *model.LLMOrgSettings {
	return &model.LLMOrgSettings{
		OrganizationID:   settings.OrganizationID.Hex(),
		PersonaKey:       settings.PersonaKey,
		DefaultModelKey:  settings.DefaultModelKey,
		FallbackModelKey: settings.FallbackModelKey,
	}
}

func toModelLLMCall(call *domain.LLMCall) *model.LLMCall {
	var profile *model.ComplianceProfile
	if call.ComplianceProfile != "" {
		p := model.ComplianceProfile(call.ComplianceProfile)
		profile = &p
	}
	return &model.LLMCall{
		ID:                      call.ID.Hex(),
		OrganizationID:          call.OrganizationID.Hex(),
		ModelKey:                call.ModelKey,
		PersonaKey:              call.PersonaKey,
		RoutingReason:           call.RoutingReason,
		FallbackUsed:            call.FallbackUsed,
		InputTokens:             call.InputTokens,
		OutputTokens:            call.OutputTokens,
		LatencyMs:               int(call.LatencyMS),
		Status:                  string(call.Status),
		ComplianceProfile:       profile,
		CompliancePolicyVersion: optionalString(call.CompliancePolicyVersion),
		ComplianceReflexVersion: optionalString(call.ComplianceReflexVersion),
		SafetyResult:            optionalString(call.SafetyResult),
		CreatedAt:               call.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func optionalString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	out := v
	return &out
}

func toModelLLMHealth(items []llmsvc.ModelHealth) []*model.LLMModelHealth {
	out := make([]*model.LLMModelHealth, 0, len(items))
	for _, item := range items {
		out = append(out, &model.LLMModelHealth{
			ModelKey:     item.ModelKey,
			DisplayName:  item.DisplayName,
			HealthStatus: item.HealthStatus,
			ProviderKey:  item.ProviderKey,
		})
	}
	return out
}

func toModelLLMUsageDashboard(dashboard *llmsvc.UsageDashboard) *model.LLMUsageDashboard {
	byModel := make([]*model.LLMModelUsage, 0, len(dashboard.ByModel))
	for _, row := range dashboard.ByModel {
		byModel = append(byModel, &model.LLMModelUsage{
			ModelKey:         row.ModelKey,
			DisplayName:      row.DisplayName,
			CallCount:        row.CallCount,
			InputTokens:      row.InputTokens,
			OutputTokens:     row.OutputTokens,
			TotalTokens:      row.InputTokens + row.OutputTokens,
			EstimatedCostUsd: row.EstimatedCostUSD,
		})
	}
	recent := make([]*model.LLMCall, 0, len(dashboard.RecentCalls))
	for i := range dashboard.RecentCalls {
		recent = append(recent, toModelLLMCall(&dashboard.RecentCalls[i]))
	}
	summary := dashboard.Summary
	return &model.LLMUsageDashboard{
		Summary: &model.LLMUsageSummary{
			CallCount:        summary.CallCount,
			InputTokens:      summary.InputTokens,
			OutputTokens:     summary.OutputTokens,
			TotalTokens:      summary.InputTokens + summary.OutputTokens,
			EstimatedCostUsd: summary.EstimatedCostUSD,
			FallbackCount:    summary.FallbackCount,
		},
		ByModel:     byModel,
		RecentCalls: recent,
	}
}

func previewContent(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 240 {
		return content
	}
	return content[:240] + "..."
}
