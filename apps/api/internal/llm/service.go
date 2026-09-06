package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	Providers   *repository.LLMProviderRepository
	Models      *repository.LLMModelRepository
	Routing     *repository.LLMRoutingPolicyRepository
	Personas    *repository.LLMPersonaRepository
	Drafts      *repository.LLMConfigurationDraftRepository
	OrgSettings *repository.LLMOrgSettingsRepository
	Snapshots   *repository.ConfigSnapshotRepository
	ConfigAudit *repository.ConfigAuditRepository
	Calls       *repository.LLMCallRepository
	Usage       *repository.LLMUsageDailyRepository
	Gateway     *Gateway
	Tenant      *tenant.Guard
	Security    *repository.SecurityEventRepository
}

type DraftInput struct {
	OrganizationID       *primitive.ObjectID
	RoutingPolicyVersion string
	PersonaKey           string
	PersonaVersion       string
	DefaultModelKey      string
	FallbackModelKey     string
	Reason               string
}

func (s *Service) ListConfigurationDrafts(ctx context.Context, actorID, orgID primitive.ObjectID, limit int64) ([]domain.LLMConfigurationDraft, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	return s.Drafts.List(ctx, &orgID, limit)
}

func (s *Service) ListProviders(ctx context.Context, actorID primitive.ObjectID, orgID *primitive.ObjectID) ([]domain.LLMProvider, error) {
	if orgID != nil {
		if _, err := s.requireRead(ctx, actorID, *orgID); err != nil {
			return nil, err
		}
	}
	return s.Providers.List(ctx)
}

func (s *Service) ListModels(ctx context.Context, actorID, orgID primitive.ObjectID) ([]domain.LLMModel, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	models, err := s.Models.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return filterModelsForOrg(models, orgID), nil
}

func (s *Service) ListRoutingPolicies(ctx context.Context, actorID, orgID primitive.ObjectID, limit int64) ([]domain.LLMRoutingPolicy, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	return s.Routing.List(ctx, &orgID, limit)
}

func (s *Service) ListPersonas(ctx context.Context, actorID, orgID primitive.ObjectID, limit int64) ([]domain.LLMPersona, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	if items, err := s.Personas.List(ctx, &orgID, limit); err == nil && len(items) > 0 {
		return items, nil
	}
	return s.Personas.List(ctx, nil, limit)
}

func (s *Service) ActiveConfiguration(ctx context.Context, actorID, orgID primitive.ObjectID) (*domain.ConfigSnapshot, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	if settings, err := s.OrgSettings.FindByOrg(ctx, orgID); err == nil && settings.ActiveConfigSnapshotID != nil {
		return s.Snapshots.FindByID(ctx, *settings.ActiveConfigSnapshotID)
	}
	return s.Snapshots.FindLatestPublished(ctx, &orgID, domain.ConfigSnapshotKindPhase6LLM)
}

func (s *Service) ConfigurationHistory(ctx context.Context, actorID, orgID primitive.ObjectID, limit int64) ([]domain.ConfigSnapshot, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	return s.Snapshots.ListPublished(ctx, &orgID, domain.ConfigSnapshotKindPhase6LLM, limit)
}

func (s *Service) CreateDraft(ctx context.Context, actorID primitive.ObjectID, in DraftInput) (*domain.LLMConfigurationDraft, error) {
	orgID := in.OrganizationID
	if orgID != nil {
		if _, err := s.requireConfigure(ctx, actorID, *orgID); err != nil {
			return nil, err
		}
	}
	draft := &domain.LLMConfigurationDraft{
		OrganizationID:       orgID,
		Status:               domain.LLMConfigDraft,
		RoutingPolicyVersion: strings.TrimSpace(in.RoutingPolicyVersion),
		PersonaKey:           strings.TrimSpace(in.PersonaKey),
		PersonaVersion:       strings.TrimSpace(in.PersonaVersion),
		DefaultModelKey:      strings.TrimSpace(in.DefaultModelKey),
		FallbackModelKey:     strings.TrimSpace(in.FallbackModelKey),
		Reason:               strings.TrimSpace(in.Reason),
		CreatedBy:            actorID,
		UpdatedBy:            actorID,
	}
	if draft.RoutingPolicyVersion == "" {
		draft.RoutingPolicyVersion = domain.LLMPlatformDefaultRoutingVersion
	}
	if draft.PersonaKey == "" {
		draft.PersonaKey = domain.PersonaCarefulAnalyst
	}
	if draft.PersonaVersion == "" {
		draft.PersonaVersion = domain.LLMPlatformDefaultPersonaVersion
	}
	if draft.DefaultModelKey == "" {
		draft.DefaultModelKey = ModelKeyCareful
	}
	if draft.FallbackModelKey == "" {
		draft.FallbackModelKey = ModelKeyResult
	}
	if err := s.Drafts.Insert(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *Service) ValidateDraft(ctx context.Context, actorID, orgID, draftID primitive.ObjectID) (*domain.LLMConfigurationDraft, error) {
	if _, err := s.requireConfigure(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	draft, err := s.loadDraftForOrg(ctx, orgID, draftID)
	if err != nil {
		return nil, err
	}
	errs := s.validateDraft(ctx, draft)
	draft.ValidationErrors = errs
	now := time.Now().UTC()
	draft.ValidatedAt = &now
	if len(errs) == 0 {
		draft.Status = domain.LLMConfigValidated
	}
	if err := s.Drafts.Update(ctx, draft); err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		return draft, ErrPublishValidation
	}
	return draft, nil
}

func (s *Service) PublishDraft(ctx context.Context, actorID, orgID, draftID primitive.ObjectID, reason string) (*domain.ConfigSnapshot, error) {
	role, err := s.requirePublish(ctx, actorID, orgID)
	if err != nil {
		return nil, err
	}
	_ = role
	draft, err := s.loadDraftForOrg(ctx, orgID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != domain.LLMConfigValidated {
		return nil, ErrDraftNotValidated
	}
	if errs := s.validateDraft(ctx, draft); len(errs) > 0 {
		return nil, ErrPublishValidation
	}

	snap := &domain.ConfigSnapshot{
		OrganizationID:          &orgID,
		SnapshotKind:            domain.ConfigSnapshotKindPhase6LLM,
		LLMRoutingPolicyVersion: draft.RoutingPolicyVersion,
		LLMPersonaKey:           draft.PersonaKey,
		LLMPersonaVersion:       draft.PersonaVersion,
		DefaultModelKey:         draft.DefaultModelKey,
		FallbackModelKey:        draft.FallbackModelKey,
		PublishedBy:             actorID,
		Reason:                  strings.TrimSpace(reason),
	}
	if err := s.Snapshots.Create(ctx, snap); err != nil {
		return nil, err
	}
	settings := &domain.LLMOrgSettings{
		OrganizationID:         orgID,
		PersonaKey:             draft.PersonaKey,
		DefaultModelKey:        draft.DefaultModelKey,
		FallbackModelKey:       draft.FallbackModelKey,
		ActiveConfigSnapshotID: &snap.ID,
		UpdatedBy:              actorID,
	}
	if err := s.OrgSettings.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	draft.Status = domain.LLMConfigPublished
	draft.PublishedSnapshotID = &snap.ID
	draft.UpdatedBy = actorID
	_ = s.Drafts.Update(ctx, draft)
	_ = s.ConfigAudit.Record(ctx, domain.ConfigAuditLog{
		ChangedBy:  actorID,
		Reason:     snap.Reason,
		OldVersion: draft.RoutingPolicyVersion,
		NewVersion: snap.LLMRoutingPolicyVersion,
		FieldDiff: map[string]string{
			"defaultModelKey":  draft.DefaultModelKey,
			"fallbackModelKey": draft.FallbackModelKey,
			"personaKey":       draft.PersonaKey,
		},
	})
	oid := orgID
	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventLLMConfigPublished,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"snapshotId": snap.ID.Hex()},
	})
	return snap, nil
}

func (s *Service) RollbackConfiguration(ctx context.Context, actorID, orgID, snapshotID primitive.ObjectID, reason string) (*domain.ConfigSnapshot, error) {
	if _, err := s.requireRollback(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	prev, err := s.Snapshots.FindByID(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	if prev.OrganizationID == nil || *prev.OrganizationID != orgID {
		return nil, ErrForbidden
	}
	newSnap := *prev
	newSnap.ID = primitive.NilObjectID
	newSnap.PublishedBy = actorID
	newSnap.Reason = strings.TrimSpace(reason)
	if err := s.Snapshots.Create(ctx, &newSnap); err != nil {
		return nil, err
	}
	settings := &domain.LLMOrgSettings{
		OrganizationID:         orgID,
		PersonaKey:             newSnap.LLMPersonaKey,
		DefaultModelKey:        newSnap.DefaultModelKey,
		FallbackModelKey:       newSnap.FallbackModelKey,
		ActiveConfigSnapshotID: &newSnap.ID,
		UpdatedBy:              actorID,
	}
	if err := s.OrgSettings.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	oid := orgID
	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventLLMConfigRollback,
		Severity:       domain.SeverityWarning,
		Details:        map[string]string{"snapshotId": newSnap.ID.Hex(), "fromSnapshotId": snapshotID.Hex()},
	})
	return &newSnap, nil
}

func (s *Service) SetOrganizationPersona(ctx context.Context, actorID, orgID primitive.ObjectID, personaKey string) (*domain.LLMOrgSettings, error) {
	if _, err := s.requireConfigure(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	personaKey = strings.TrimSpace(personaKey)
	if personaKey != domain.PersonaCarefulAnalyst && personaKey != domain.PersonaResultAnalyst {
		return nil, ErrInvalidInput
	}
	if _, err := s.Personas.FindPublished(ctx, nil, personaKey, domain.LLMPlatformDefaultPersonaVersion); err != nil {
		return nil, err
	}
	settings := &domain.LLMOrgSettings{OrganizationID: orgID, UpdatedBy: actorID, PersonaKey: personaKey}
	if existing, err := s.OrgSettings.FindByOrg(ctx, orgID); err == nil {
		settings.DefaultModelKey = existing.DefaultModelKey
		settings.FallbackModelKey = existing.FallbackModelKey
		settings.ActiveConfigSnapshotID = existing.ActiveConfigSnapshotID
	}
	if err := s.OrgSettings.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *Service) SetOrganizationModels(ctx context.Context, actorID, orgID primitive.ObjectID, defaultModelKey, fallbackModelKey string) (*domain.LLMOrgSettings, error) {
	if _, err := s.requireConfigure(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	defaultModelKey = strings.TrimSpace(defaultModelKey)
	fallbackModelKey = strings.TrimSpace(fallbackModelKey)
	if defaultModelKey == "" || fallbackModelKey == "" {
		return nil, ErrInvalidInput
	}
	if defaultModelKey == fallbackModelKey {
		return nil, fmt.Errorf("default and fallback model must differ")
	}
	if _, err := s.Models.FindByKey(ctx, defaultModelKey); err != nil {
		return nil, err
	}
	if _, err := s.Models.FindByKey(ctx, fallbackModelKey); err != nil {
		return nil, err
	}

	settings := &domain.LLMOrgSettings{
		OrganizationID:   orgID,
		UpdatedBy:        actorID,
		DefaultModelKey:  defaultModelKey,
		FallbackModelKey: fallbackModelKey,
	}
	if existing, err := s.OrgSettings.FindByOrg(ctx, orgID); err == nil {
		settings.PersonaKey = existing.PersonaKey
		settings.ActiveConfigSnapshotID = existing.ActiveConfigSnapshotID
	}
	if err := s.OrgSettings.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *Service) GetOrgSettings(ctx context.Context, actorID, orgID primitive.ObjectID) (*domain.LLMOrgSettings, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	settings, err := s.OrgSettings.FindByOrg(ctx, orgID)
	if err != nil {
		if err == repository.ErrNotFound {
			return &domain.LLMOrgSettings{OrganizationID: orgID}, nil
		}
		return nil, err
	}
	return settings, nil
}

type UsageDashboard struct {
	Summary     UsageDashboardSummary
	ByModel     []UsageDashboardModel
	RecentCalls []domain.LLMCall
}

type UsageDashboardSummary struct {
	CallCount        int
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
	FallbackCount    int
}

type UsageDashboardModel struct {
	ModelKey         string
	DisplayName      string
	CallCount        int
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
}

func (s *Service) UsageDashboard(ctx context.Context, actorID, orgID primitive.ObjectID, fromDate string, recentLimit int) (*UsageDashboard, error) {
	if _, err := s.requireUsageRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	callCount, inputTokens, outputTokens, cost, err := s.Usage.SumSince(ctx, orgID, fromDate)
	if err != nil {
		return nil, err
	}
	fromTime, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return nil, ErrInvalidInput
	}
	aggregates, err := s.Calls.AggregateUsageByModel(ctx, orgID, fromTime)
	if err != nil {
		return nil, err
	}
	recent, err := s.Calls.ListRecent(ctx, orgID, recentLimit)
	if err != nil {
		return nil, err
	}
	models, err := s.Models.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	displayNames := map[string]string{}
	for _, m := range models {
		displayNames[m.ModelKey] = m.DisplayName
	}
	byModel := make([]UsageDashboardModel, 0, len(aggregates))
	fallbackCount := 0
	for _, row := range aggregates {
		byModel = append(byModel, UsageDashboardModel{
			ModelKey:         row.ModelKey,
			DisplayName:      displayNames[row.ModelKey],
			CallCount:        row.CallCount,
			InputTokens:      row.InputTokens,
			OutputTokens:     row.OutputTokens,
			EstimatedCostUSD: row.EstimatedCostUSD,
		})
	}
	for _, call := range recent {
		if call.FallbackUsed {
			fallbackCount++
		}
	}
	return &UsageDashboard{
		Summary: UsageDashboardSummary{
			CallCount:        callCount,
			InputTokens:      inputTokens,
			OutputTokens:     outputTokens,
			EstimatedCostUSD: cost,
			FallbackCount:    fallbackCount,
		},
		ByModel:     byModel,
		RecentCalls: recent,
	}, nil
}

func (s *Service) TestConfiguration(ctx context.Context, actorID, orgID primitive.ObjectID, personaKey, prompt string) (GatewayResponse, error) {
	if _, err := s.requireTest(ctx, actorID, orgID); err != nil {
		return GatewayResponse{}, err
	}
	return s.Gateway.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &actorID,
		CorrelationID:  primitive.NewObjectID().Hex(),
		TaskType:       "analysis",
		PersonaKey:     personaKey,
		UserPrompt:     prompt,
	})
}

func (s *Service) UsageSummary(ctx context.Context, actorID, orgID primitive.ObjectID, fromDate string) (int, int, int, float64, error) {
	if _, err := s.requireUsageRead(ctx, actorID, orgID); err != nil {
		return 0, 0, 0, 0, err
	}
	return s.Usage.SumSince(ctx, orgID, fromDate)
}

func (s *Service) GetCall(ctx context.Context, actorID, orgID, callID primitive.ObjectID) (*domain.LLMCall, error) {
	if _, err := s.requireAuditRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	return s.Calls.FindByID(ctx, orgID, callID)
}

func (s *Service) Health(ctx context.Context, actorID, orgID primitive.ObjectID) ([]ModelHealth, error) {
	if _, err := s.requireRead(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	return s.Gateway.Health(ctx)
}

func (s *Service) validateDraft(ctx context.Context, draft *domain.LLMConfigurationDraft) []string {
	var errs []string
	if _, err := s.Routing.FindPublished(ctx, nil, draft.RoutingPolicyVersion); err != nil {
		errs = append(errs, "routing policy not published")
	}
	if _, err := s.Personas.FindPublished(ctx, nil, draft.PersonaKey, draft.PersonaVersion); err != nil {
		errs = append(errs, "persona not published")
	}
	if _, err := s.Models.FindByKey(ctx, draft.DefaultModelKey); err != nil {
		errs = append(errs, "default model missing")
	}
	if _, err := s.Models.FindByKey(ctx, draft.FallbackModelKey); err != nil {
		errs = append(errs, "fallback model missing")
	}
	if draft.DefaultModelKey == draft.FallbackModelKey {
		errs = append(errs, "default and fallback must differ")
	}
	if detectExternalInjection(draft.Reason) {
		errs = append(errs, "reason contains blocked content")
	}
	return errs
}

func (s *Service) loadDraftForOrg(ctx context.Context, orgID, draftID primitive.ObjectID) (*domain.LLMConfigurationDraft, error) {
	draft, err := s.Drafts.FindByID(ctx, draftID)
	if err != nil {
		return nil, err
	}
	if draft.OrganizationID == nil || *draft.OrganizationID != orgID {
		return nil, ErrForbidden
	}
	return draft, nil
}

func (s *Service) requireRead(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return "", ErrForbidden
	}
	if !rbac.CanReadLLM(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requireConfigure(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireRead(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanConfigureLLM(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requirePublish(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireConfigure(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanPublishLLM(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requireRollback(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireRead(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanRollbackLLM(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requireTest(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireConfigure(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanTestLLM(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requireUsageRead(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireRead(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanReadLLMUsage(role) {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *Service) requireAuditRead(ctx context.Context, actorID, orgID primitive.ObjectID) (domain.OrgRole, error) {
	role, err := s.requireRead(ctx, actorID, orgID)
	if err != nil {
		return "", err
	}
	if !rbac.CanReadLLMAudit(role) {
		return "", ErrForbidden
	}
	return role, nil
}
