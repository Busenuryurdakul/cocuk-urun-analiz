package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GatewayRequest struct {
	OrganizationID       primitive.ObjectID
	UserID               *primitive.ObjectID
	AnalysisRunID        *primitive.ObjectID
	CorrelationID        string
	IdempotencyKey       string
	TaskType             string
	SafetyRisk           string
	RequireEvidence      bool
	PersonaKey           string
	UserPrompt           string
	ExternalContent      []string
	RoutingPolicyVersion string
	ConfigSnapshotID     *primitive.ObjectID
}

type GatewayResponse struct {
	CallID                  string  `json:"callId"`
	Content                 string  `json:"content"`
	ModelKey                string  `json:"modelKey"`
	ProviderKey             string  `json:"providerKey"`
	FallbackUsed            bool    `json:"fallbackUsed"`
	EscalationUsed          bool    `json:"escalationUsed"`
	RoutingReason           string  `json:"routingReason"`
	PersonaKey              string  `json:"personaKey"`
	PersonaVersion          string  `json:"personaVersion"`
	CorrelationID           string  `json:"correlationId"`
	InputTokens             int     `json:"inputTokens"`
	OutputTokens            int     `json:"outputTokens"`
	LatencyMS               int64   `json:"latencyMs"`
	EstimatedCostUSD        float64 `json:"estimatedCostUsd"`
	RedactionApplied        bool    `json:"redactionApplied"`
	ComplianceProfile       string  `json:"complianceProfile,omitempty"`
	CompliancePolicyVersion string  `json:"compliancePolicyVersion,omitempty"`
	ComplianceReflexVersion string  `json:"complianceReflexVersion,omitempty"`
	SafetyResult            string  `json:"safetyResult,omitempty"`
}

type GatewayDeps struct {
	Orgs        *repository.OrganizationRepository
	OrgSettings *repository.LLMOrgSettingsRepository
	Providers   *repository.LLMProviderRepository
	Models      *repository.LLMModelRepository
	Routing     *repository.LLMRoutingPolicyRepository
	Personas    *repository.LLMPersonaRepository
	Calls       *repository.LLMCallRepository
	Usage       *repository.LLMUsageDailyRepository
	Router      *Router
	Enforcer    *Enforcer
	Mock        Provider
	HTTP        Provider
	UseMock     bool
	MaxRetries  int
	Timeout     time.Duration
}

type Gateway struct {
	deps GatewayDeps
}

func NewGateway(deps GatewayDeps) *Gateway {
	if deps.MaxRetries <= 0 {
		deps.MaxRetries = 1
	}
	if deps.Timeout <= 0 {
		deps.Timeout = 30 * time.Second
	}
	return &Gateway{deps: deps}
}

func (g *Gateway) Complete(ctx context.Context, req GatewayRequest) (GatewayResponse, error) {
	if req.IdempotencyKey != "" {
		if existing, err := g.deps.Calls.FindByIdempotency(ctx, req.OrganizationID, req.IdempotencyKey); err == nil {
			return gatewayResponseFromCall(existing), nil
		}
	}

	org, err := g.deps.Orgs.FindByID(ctx, req.OrganizationID)
	if err != nil {
		return GatewayResponse{}, err
	}

	personaKey := strings.TrimSpace(req.PersonaKey)
	if personaKey == "" {
		if settings, err := g.deps.OrgSettings.FindByOrg(ctx, req.OrganizationID); err == nil && settings.PersonaKey != "" {
			personaKey = settings.PersonaKey
		}
	}
	if personaKey == "" {
		personaKey = domain.PersonaCarefulAnalyst
	}

	persona, err := g.resolvePersona(ctx, req.OrganizationID, personaKey)
	if err != nil {
		return GatewayResponse{}, err
	}

	userID := primitive.NilObjectID
	if req.UserID != nil {
		userID = *req.UserID
	}

	enforced, err := g.deps.Enforcer.PreCall(ctx, org, EnforcementInput{
		OrganizationID:     req.OrganizationID,
		UserID:             userID,
		ConfigSnapshotID:   req.ConfigSnapshotID,
		UserPrompt:         req.UserPrompt,
		ExternalContent:    req.ExternalContent,
		PersonaInstruction: persona.SystemInstruction,
	})
	if err != nil {
		return g.recordBlocked(ctx, req, persona, enforced.Compliance, err)
	}
	if enforced.Blocked {
		return g.recordBlocked(ctx, req, persona, enforced.Compliance, ErrComplianceBlocked)
	}

	systemInstruction := BuildSystemInstruction(persona.SystemInstruction, enforced.Compliance.ReflexInstruction)

	policyVersion := strings.TrimSpace(req.RoutingPolicyVersion)
	if policyVersion == "" {
		policyVersion = domain.LLMPlatformDefaultRoutingVersion
	}
	policy, err := g.deps.Routing.FindPublished(ctx, nil, policyVersion)
	if err != nil {
		if settings, sErr := g.deps.OrgSettings.FindByOrg(ctx, req.OrganizationID); sErr == nil {
			_ = settings
		}
		policy, err = g.deps.Routing.FindPublished(ctx, &req.OrganizationID, policyVersion)
		if err != nil {
			return GatewayResponse{}, ErrInvalidPolicyVersion
		}
	}

	models, err := g.deps.Models.ListActive(ctx)
	if err != nil {
		return GatewayResponse{}, err
	}
	models = filterModelsForOrg(models, req.OrganizationID)
	if len(models) < 2 {
		return GatewayResponse{}, ErrInsufficientModels
	}

	decision, err := g.deps.Router.Decide(ctx, RouteRequest{
		TaskType:        req.TaskType,
		SafetyRisk:      req.SafetyRisk,
		RequireEvidence: req.RequireEvidence,
		PersonaKey:      personaKey,
		Policy:          policy,
		Models:          models,
	})
	if err != nil {
		return GatewayResponse{}, err
	}

	primaryModel, err := g.deps.Models.FindByKey(ctx, decision.PrimaryModelKey)
	if err != nil {
		return GatewayResponse{}, err
	}
	fallbackModel, err := g.deps.Models.FindByKey(ctx, decision.FallbackModelKey)
	if err != nil {
		return GatewayResponse{}, err
	}

	g.deps.Router.IncLoad(ctx, primaryModel.ModelKey)
	defer g.deps.Router.DecLoad(ctx, primaryModel.ModelKey)

	provider := g.selectProvider()
	start := time.Now()
	result, usedFallback, escalationUsed, retryCount, invokeErr := g.invokeWithQualityEscalation(
		ctx,
		provider,
		primaryModel,
		fallbackModel,
		systemInstruction,
		enforced.UserPrompt,
		req.TaskType,
	)
	latency := time.Since(start).Milliseconds()
	if invokeErr != nil {
		call := g.buildCall(req, persona, policy.Version, decision, primaryModel.ModelKey, usedFallback, retryCount, 0, 0, latency, domain.LLMCallFailed, enforced, invokeErr.Error())
		_ = g.persistCall(ctx, call)
		return GatewayResponse{}, invokeErr
	}

	postResult, postErr := g.deps.Enforcer.PostCall(ctx, org, userID, result.Content, enforced.Compliance)
	if postErr != nil {
		call := g.buildCall(req, persona, policy.Version, decision, primaryModel.ModelKey, usedFallback, retryCount, result.InputTokens, result.OutputTokens, latency, domain.LLMCallBlocked, enforced, postErr.Error())
		call.OutputHash = HashContent(result.Content)
		call.SafetyResult = postResult.SafetyResult
		_ = g.persistCall(ctx, call)
		return GatewayResponse{}, postErr
	}

	selectedKey := primaryModel.ModelKey
	if usedFallback {
		selectedKey = fallbackModel.ModelKey
	}
	cost := estimateCost(result.InputTokens, result.OutputTokens)
	reason := decision.Reason
	if escalationUsed {
		reason += ";quality_escalation=true"
	}
	call := g.buildCall(req, persona, policy.Version, decision, selectedKey, usedFallback, retryCount, result.InputTokens, result.OutputTokens, latency, domain.LLMCallSucceeded, enforced, "")
	call.OutputHash = HashContent(result.Content)
	call.RoutingReason = reason
	call.SafetyResult = postResult.SafetyResult
	_ = g.persistCall(ctx, call)

	return GatewayResponse{
		CallID:                  call.ID.Hex(),
		Content:                 result.Content,
		ModelKey:                selectedKey,
		ProviderKey:             call.ProviderKey,
		FallbackUsed:            usedFallback,
		EscalationUsed:          escalationUsed,
		RoutingReason:           reason,
		PersonaKey:              persona.PersonaKey,
		PersonaVersion:          persona.Version,
		CorrelationID:           req.CorrelationID,
		InputTokens:             result.InputTokens,
		OutputTokens:            result.OutputTokens,
		LatencyMS:               latency,
		EstimatedCostUSD:        cost,
		RedactionApplied:        enforced.RedactionApplied,
		ComplianceProfile:       string(enforced.Compliance.Profile),
		CompliancePolicyVersion: enforced.Compliance.PolicyVersion,
		ComplianceReflexVersion: enforced.Compliance.ReflexVersion,
		SafetyResult:            postResult.SafetyResult,
	}, nil
}

func (g *Gateway) invokeWithFallback(ctx context.Context, provider Provider, primary, fallback *domain.LLMModel, systemInstruction, userPrompt string) (CompletionResult, bool, int, error) {
	var lastErr error
	retries := 0
	for attempt := 0; attempt <= g.deps.MaxRetries; attempt++ {
		res, err := g.invokeModel(ctx, provider, primary, systemInstruction, userPrompt)
		if err == nil {
			return res, false, retries, nil
		}
		lastErr = err
		retries++
	}
	res, err := g.invokeModel(ctx, provider, fallback, systemInstruction, userPrompt)
	if err == nil {
		return res, true, retries, nil
	}
	if lastErr != nil {
		return CompletionResult{}, true, retries, lastErr
	}
	return CompletionResult{}, true, retries, err
}

func (g *Gateway) invokeWithQualityEscalation(
	ctx context.Context,
	provider Provider,
	primary, fallback *domain.LLMModel,
	systemInstruction, userPrompt, taskType string,
) (CompletionResult, bool, bool, int, error) {
	result, usedFallback, retries, err := g.invokeWithFallback(ctx, provider, primary, fallback, systemInstruction, userPrompt)
	if err != nil {
		return CompletionResult{}, usedFallback, false, retries, err
	}
	if usedFallback || !ShouldEscalateQuality(taskType, userPrompt, result.Content) {
		return result, usedFallback, false, retries, nil
	}

	escalated, escErr := g.invokeModel(ctx, provider, fallback, systemInstruction, userPrompt)
	if escErr != nil {
		return result, false, false, retries, nil
	}
	return escalated, true, true, retries, nil
}

func (g *Gateway) invokeModel(ctx context.Context, provider Provider, model *domain.LLMModel, systemInstruction, userPrompt string) (CompletionResult, error) {
	providerDoc, err := g.deps.Providers.FindByKey(ctx, providerKeyForModel(model.ModelKey))
	if err != nil {
		return CompletionResult{}, err
	}
	baseURL := ResolveBaseURLRef(providerDoc.BaseURLRef)
	apiKey := ResolveSecretRef(providerDoc.SecretRef)
	if !g.deps.UseMock && !ProviderCredentialsConfigured(baseURL, apiKey) {
		return CompletionResult{}, fmt.Errorf("provider credentials not configured for %s", model.ModelKey)
	}
	return provider.Complete(ctx, CompletionRequest{
		ModelKey:          model.ModelKey,
		ProviderModelName: model.ProviderModelName,
		SystemInstruction: systemInstruction,
		UserPrompt:        userPrompt,
		Timeout:           g.deps.Timeout,
		BaseURL:           baseURL,
		APIKey:            apiKey,
	})
}

func providerKeyForModel(modelKey string) string {
	if modelKey == ModelKeyResult {
		return ProviderKeySecondary
	}
	return ProviderKeyPrimary
}

func (g *Gateway) selectProvider() Provider {
	if g.deps.UseMock {
		return g.deps.Mock
	}
	if g.deps.HTTP != nil {
		return g.deps.HTTP
	}
	return g.deps.Mock
}

func (g *Gateway) resolvePersona(ctx context.Context, orgID primitive.ObjectID, personaKey string) (*domain.LLMPersona, error) {
	if p, err := g.deps.Personas.FindPublished(ctx, &orgID, personaKey, domain.LLMPlatformDefaultPersonaVersion); err == nil {
		return p, nil
	}
	return g.deps.Personas.FindPublished(ctx, nil, personaKey, domain.LLMPlatformDefaultPersonaVersion)
}

func filterModelsForOrg(models []domain.LLMModel, orgID primitive.ObjectID) []domain.LLMModel {
	out := make([]domain.LLMModel, 0, len(models))
	for _, m := range models {
		if len(m.OrganizationAllowlist) == 0 {
			out = append(out, m)
			continue
		}
		for _, allowed := range m.OrganizationAllowlist {
			if allowed == orgID {
				out = append(out, m)
				break
			}
		}
	}
	return out
}

func (g *Gateway) buildCall(req GatewayRequest, persona *domain.LLMPersona, policyVersion string, decision RouteDecision, modelKey string, fallbackUsed bool, retryCount, inTokens, outTokens int, latency int64, status domain.LLMCallStatus, enforced EnforcementResult, errCode string) *domain.LLMCall {
	call := &domain.LLMCall{
		OrganizationID:          req.OrganizationID,
		UserID:                  req.UserID,
		AnalysisRunID:           req.AnalysisRunID,
		CorrelationID:           req.CorrelationID,
		IdempotencyKey:          req.IdempotencyKey,
		ProviderKey:             providerKeyForModel(modelKey),
		ModelKey:                modelKey,
		PersonaKey:              persona.PersonaKey,
		PersonaVersion:          persona.Version,
		ConfigSnapshotID:        req.ConfigSnapshotID,
		RoutingPolicyVersion:    policyVersion,
		RoutingReason:           decision.Reason,
		PrimaryModelKey:         decision.PrimaryModelKey,
		FallbackUsed:            fallbackUsed,
		RetryCount:              retryCount,
		InputTokens:             inTokens,
		OutputTokens:            outTokens,
		LatencyMS:               latency,
		EstimatedCostUSD:        estimateCost(inTokens, outTokens),
		Status:                  status,
		RedactionApplied:        enforced.RedactionApplied,
		ComplianceProfile:       string(enforced.Compliance.Profile),
		CompliancePolicyVersion: enforced.Compliance.PolicyVersion,
		ComplianceReflexVersion: enforced.Compliance.ReflexVersion,
		InputHash:               HashContent(req.UserPrompt),
		ErrorCode:               errCode,
	}
	return call
}

func (g *Gateway) persistCall(ctx context.Context, call *domain.LLMCall) error {
	if err := g.deps.Calls.Insert(ctx, call); err != nil {
		return err
	}
	date := call.CreatedAt.UTC().Format("2006-01-02")
	return g.deps.Usage.Increment(ctx, call.OrganizationID, date, call.InputTokens, call.OutputTokens, call.EstimatedCostUSD, call.FallbackUsed)
}

func (g *Gateway) recordBlocked(ctx context.Context, req GatewayRequest, persona *domain.LLMPersona, complianceCtx ComplianceContext, blockErr error) (GatewayResponse, error) {
	call := &domain.LLMCall{
		OrganizationID:          req.OrganizationID,
		UserID:                  req.UserID,
		AnalysisRunID:           req.AnalysisRunID,
		CorrelationID:           req.CorrelationID,
		IdempotencyKey:          req.IdempotencyKey,
		ConfigSnapshotID:        req.ConfigSnapshotID,
		PersonaKey:              persona.PersonaKey,
		PersonaVersion:          persona.Version,
		Status:                  domain.LLMCallBlocked,
		InputHash:               HashContent(req.UserPrompt),
		ErrorCode:               blockErr.Error(),
		SafetyResult:            "blocked",
		ComplianceProfile:       string(complianceCtx.Profile),
		CompliancePolicyVersion: complianceCtx.PolicyVersion,
		ComplianceReflexVersion: complianceCtx.ReflexVersion,
	}
	_ = g.deps.Calls.Insert(ctx, call)
	return GatewayResponse{}, blockErr
}

func gatewayResponseFromCall(call *domain.LLMCall) GatewayResponse {
	return GatewayResponse{
		CallID:                  call.ID.Hex(),
		ModelKey:                call.ModelKey,
		ProviderKey:             call.ProviderKey,
		FallbackUsed:            call.FallbackUsed,
		RoutingReason:           call.RoutingReason,
		PersonaKey:              call.PersonaKey,
		PersonaVersion:          call.PersonaVersion,
		CorrelationID:           call.CorrelationID,
		InputTokens:             call.InputTokens,
		OutputTokens:            call.OutputTokens,
		LatencyMS:               call.LatencyMS,
		EstimatedCostUSD:        call.EstimatedCostUSD,
		RedactionApplied:        call.RedactionApplied,
		ComplianceProfile:       call.ComplianceProfile,
		CompliancePolicyVersion: call.CompliancePolicyVersion,
		ComplianceReflexVersion: call.ComplianceReflexVersion,
		SafetyResult:            call.SafetyResult,
	}
}

func estimateCost(inputTokens, outputTokens int) float64 {
	return float64(inputTokens)*0.000001 + float64(outputTokens)*0.000002
}

func (g *Gateway) Health(ctx context.Context) ([]ModelHealth, error) {
	models, err := g.deps.Models.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := g.deps.Providers.List(ctx)
	if err != nil {
		return nil, err
	}
	providerByID := map[primitive.ObjectID]domain.LLMProvider{}
	for _, p := range providers {
		providerByID[p.ID] = p
	}
	out := make([]ModelHealth, 0, len(models))
	for _, m := range models {
		p := providerByID[m.ProviderID]
		status := m.HealthStatus
		if !g.deps.UseMock {
			err := g.deps.HTTP.HealthCheck(ctx, ResolveBaseURLRef(p.BaseURLRef), ResolveSecretRef(p.SecretRef), m.ProviderModelName)
			if err != nil {
				status = domain.LLMHealthUnhealthy
			} else {
				status = domain.LLMHealthHealthy
			}
			_ = g.deps.Models.UpdateHealth(ctx, m.ModelKey, status)
		}
		out = append(out, ModelHealth{
			ModelKey:     m.ModelKey,
			DisplayName:  m.DisplayName,
			HealthStatus: string(status),
			ProviderKey:  p.ProviderKey,
		})
	}
	return out, nil
}

type ModelHealth struct {
	ModelKey     string `json:"modelKey"`
	DisplayName  string `json:"displayName"`
	HealthStatus string `json:"healthStatus"`
	ProviderKey  string `json:"providerKey"`
}
