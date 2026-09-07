package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/analysis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	DefaultMaxToolLoopIterations = 10
	DefaultTotalRunTimeout       = 10 * time.Minute
)

type Service struct {
	Runs          *repository.AnalysisRunRepository
	Events        *repository.AgentRunEventRepository
	Executions    *repository.ToolExecutionRepository
	Snapshots     *repository.ConfigSnapshotRepository
	Orgs          *repository.OrganizationRepository
	Resolver      *InputResolver
	Executor      *Executor
	Authorizer    *Authorizer
	Compliance    *compliance.Engine
	Registry      *Registry
	Orchestrator  *OrchestratorClient
	Grants        *GrantStore
	Leases        *LeaseStore
	Security      *repository.SecurityEventRepository
	Tenant        *tenant.Guard
	Analysis      *analysis.Service
	grantMu       sync.Mutex
	resolvedCache map[string]ResolvedInput
}

type StartRunInput struct {
	OrganizationID         primitive.ObjectID
	ActorID                primitive.ObjectID
	ProductID              primitive.ObjectID
	MarketplaceImportRunID *primitive.ObjectID
	ClientRequestID        string
}

func NewService(deps ServiceDeps) *Service {
	s := &Service{
		Runs:          deps.Runs,
		Events:        deps.Events,
		Executions:    deps.Executions,
		Snapshots:     deps.Snapshots,
		Orgs:          deps.Orgs,
		Resolver:      deps.Resolver,
		Executor:      deps.Executor,
		Authorizer:    deps.Authorizer,
		Compliance:    deps.Compliance,
		Registry:      deps.Registry,
		Orchestrator:  deps.Orchestrator,
		Grants:        deps.Grants,
		Leases:        deps.Leases,
		Security:      deps.Security,
		Tenant:        deps.Tenant,
		Analysis:      deps.Analysis,
		resolvedCache: make(map[string]ResolvedInput),
	}
	if s.Registry == nil {
		s.Registry = NewRegistry()
	}
	return s
}

type ServiceDeps struct {
	Runs         *repository.AnalysisRunRepository
	Events       *repository.AgentRunEventRepository
	Executions   *repository.ToolExecutionRepository
	Snapshots    *repository.ConfigSnapshotRepository
	Orgs         *repository.OrganizationRepository
	Resolver     *InputResolver
	Executor     *Executor
	Authorizer   *Authorizer
	Compliance   *compliance.Engine
	Registry     *Registry
	Orchestrator *OrchestratorClient
	Grants       *GrantStore
	Leases       *LeaseStore
	Security     *repository.SecurityEventRepository
	Tenant       *tenant.Guard
	Analysis     *analysis.Service
}

func (s *Service) StartRun(ctx context.Context, input StartRunInput) (*domain.AnalysisRun, error) {
	if input.ClientRequestID == "" {
		return nil, ErrInvalidInput
	}

	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanStartAnalysisRun(role) {
		return nil, ErrForbidden
	}

	existing, err := s.Runs.FindByClientRequestID(ctx, input.OrganizationID, input.ClientRequestID)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if err := s.Resolver.ValidateProvenance(ctx, input.OrganizationID, input.MarketplaceImportRunID, input.ProductID); err != nil {
		return nil, err
	}
	if _, err := s.Resolver.Products.FindByID(ctx, input.OrganizationID, input.ProductID); err != nil {
		return nil, err
	}

	pre, err := s.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "start_analysis_run",
		OrganizationID: input.OrganizationID,
		UserID:         input.ActorID,
		Purpose:        domain.ConsentPurposeDataProcessing,
		Fields:         map[string]string{"productId": input.ProductID.Hex()},
	})
	if err != nil {
		return nil, err
	}
	if !pre.Allowed {
		return nil, ErrComplianceRejected
	}

	org, err := s.Orgs.FindByID(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	snap, err := s.ensureSnapshot(ctx, input.OrganizationID, input.ActorID, org.CompliancePolicyVersion)
	if err != nil {
		return nil, err
	}

	traceID := uuid.NewString()
	run := &domain.AnalysisRun{
		OrganizationID:           input.OrganizationID,
		CreatedByUserID:          input.ActorID,
		ProductID:                input.ProductID,
		MarketplaceImportRunID:   input.MarketplaceImportRunID,
		ClientRequestID:          input.ClientRequestID,
		Status:                   domain.AnalysisStatusPending,
		TraceID:                  traceID,
		CorrelationID:            traceID,
		ConfigSnapshotID:         snap.ID,
		ToolRegistryVersion:      snap.ToolRegistryVersion,
		ToolPolicyVersion:        snap.ToolPolicyVersion,
		CompliancePolicyVersion:  org.CompliancePolicyVersion,
		PlannerVersion:           snap.PlannerVersion,
		ObservationSchemaVersion: snap.ObservationSchemaVersion,
	}
	if err := s.Runs.Create(ctx, run); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return s.Runs.FindByClientRequestID(ctx, input.OrganizationID, input.ClientRequestID)
		}
		return nil, err
	}

	resolved, err := s.Resolver.Resolve(ctx, input.OrganizationID, input.ProductID, input.MarketplaceImportRunID)
	if err != nil {
		_ = s.dispatchFailure(ctx, run, err.Error())
		return nil, err
	}
	s.cacheResolved(run.ID.Hex(), *resolved)

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventAnalysisRunStarted,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"analysisRunId": run.ID.Hex(), "productId": input.ProductID.Hex()},
	})

	caps := s.BuildCapabilitySnapshot()
	if s.Orchestrator == nil {
		_ = s.dispatchFailure(ctx, run, ErrOrchestratorUnavailable.Error())
		return nil, ErrOrchestratorUnavailable
	}
	if err := s.Orchestrator.StartAnalysisRun(ctx, StartAnalysisRunRequest{
		OrganizationID: input.OrganizationID.Hex(),
		AnalysisRunID:  run.ID.Hex(),
		ProductID:      input.ProductID.Hex(),
		TraceID:        traceID,
		ActorUserID:    input.ActorID.Hex(),
		Capabilities:   caps,
	}); err != nil {
		_ = s.dispatchFailure(ctx, run, err.Error())
		return nil, err
	}

	return run, nil
}

func (s *Service) CancelRun(ctx context.Context, actorID, organizationID, runID primitive.ObjectID) (*domain.AnalysisRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}

	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return nil, err
	}
	isOwner := run.CreatedByUserID == actorID
	if !rbac.CanCancelAnalysisRun(role, isOwner) {
		return nil, ErrForbidden
	}
	if run.Status == domain.AnalysisStatusCompleted || run.Status == domain.AnalysisStatusFailed || run.Status == domain.AnalysisStatusRejected {
		return nil, ErrRunNotCancellable
	}

	run.CancellationRequested = true
	cancelledBy := actorID
	run.CancelledByUserID = &cancelledBy
	run.TerminalReason = "CANCELLED"
	if err := s.Runs.Update(ctx, organizationID, run); err != nil {
		return nil, err
	}

	if s.Orchestrator != nil {
		_ = s.Orchestrator.CancelRun(ctx, CancelRunRequest{
			OrganizationID: organizationID.Hex(),
			AnalysisRunID:  runID.Hex(),
			TraceID:        run.TraceID,
		})
	}

	uid := actorID
	oid := organizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventAnalysisRunCancelled,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"analysisRunId": run.ID.Hex()},
	})
	return run, nil
}

func (s *Service) GetRun(ctx context.Context, actorID, organizationID, runID primitive.ObjectID) (*domain.AnalysisRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	return s.Runs.FindByID(ctx, organizationID, runID)
}

func (s *Service) ListRuns(ctx context.Context, actorID, organizationID primitive.ObjectID, status *domain.AnalysisRunStatus, limit int) ([]domain.AnalysisRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	return s.Runs.ListByOrg(ctx, organizationID, status, limit)
}

func (s *Service) ListEvents(ctx context.Context, actorID, organizationID, runID primitive.ObjectID, afterSequence int64, limit int) ([]domain.AgentRunEvent, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	if _, err := s.Runs.FindByID(ctx, organizationID, runID); err != nil {
		return nil, err
	}
	return s.Events.ListByRun(ctx, organizationID, runID, afterSequence, limit)
}

// ProcessRun is intentionally removed from Go lifecycle ownership.
// Python RunManager is the sole orchestrator; Go exposes internal APIs only.

func (s *Service) ensureSnapshot(ctx context.Context, orgID, actorID primitive.ObjectID, complianceVersion string) (*domain.ConfigSnapshot, error) {
	orgCopy := orgID
	snap := &domain.ConfigSnapshot{
		OrganizationID:           &orgCopy,
		SnapshotKind:             domain.ConfigSnapshotKindPhase5,
		ToolRegistryVersion:      RegistryVersion(s.Registry),
		ToolPolicyVersion:        ToolPolicyVersion(s.Registry),
		CompliancePolicyVersion:  complianceVersion,
		PlannerVersion:           PythonPlannerVersion(),
		ObservationSchemaVersion: ObservationSchemaVersion(),
		RuntimeConfig:            map[string]any{"maxToolLoopIterations": DefaultMaxToolLoopIterations},
		PublishedBy:              actorID,
		Reason:                   "phase5 analysis run bootstrap",
	}
	if err := s.Snapshots.Create(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Service) dispatchFailure(ctx context.Context, run *domain.AnalysisRun, reason string) error {
	now := time.Now().UTC()
	run.Status = domain.AnalysisStatusFailed
	run.CurrentPhase = domain.AgentPhaseRunFailed
	run.TerminalError = compliance.SanitizeForAudit(reason)
	run.TerminalReason = "ORCHESTRATOR_DISPATCH_FAILED"
	run.CompletedAt = &now
	_ = s.Runs.Update(ctx, run.OrganizationID, run)
	return fmt.Errorf("%s", reason)
}

func (s *Service) cacheResolved(runID string, resolved ResolvedInput) {
	s.grantMu.Lock()
	defer s.grantMu.Unlock()
	s.resolvedCache[runID] = resolved
}

func (s *Service) getResolved(runID string) (ResolvedInput, bool) {
	s.grantMu.Lock()
	defer s.grantMu.Unlock()
	v, ok := s.resolvedCache[runID]
	return v, ok
}

// LegacyQueueRemoved documents that analysis:run:queue is not used by Go consumer anymore.
var _ = queue.AnalysisQueueKey
