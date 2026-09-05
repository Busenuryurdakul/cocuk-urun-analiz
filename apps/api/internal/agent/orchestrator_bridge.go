package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const maxEventMetadataBytes = 4096

type RunContextResponse struct {
	OrganizationID string             `json:"organizationId"`
	AnalysisRunID  string             `json:"analysisRunId"`
	ProductID      string             `json:"productId"`
	TraceID        string             `json:"traceId"`
	ActorUserID    string             `json:"actorUserId"`
	Status         string             `json:"status"`
	Capabilities   CapabilitySnapshot `json:"capabilities"`
	ReviewCount    int                `json:"marketplaceReviewCount"`
	UGCCount       int                `json:"ugcCount"`
}

func (s *Service) RunContext(ctx context.Context, organizationID, runID primitive.ObjectID) (*RunContextResponse, error) {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return nil, err
	}
	resolved, ok := s.getResolved(runID.Hex())
	if !ok {
		resolvedPtr, resolveErr := s.Resolver.Resolve(ctx, organizationID, run.ProductID, run.MarketplaceImportRunID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		resolved = *resolvedPtr
		s.cacheResolved(runID.Hex(), resolved)
	}
	return &RunContextResponse{
		OrganizationID: organizationID.Hex(),
		AnalysisRunID:  runID.Hex(),
		ProductID:      run.ProductID.Hex(),
		TraceID:        run.TraceID,
		ActorUserID:    run.CreatedByUserID.Hex(),
		Status:         string(run.Status),
		Capabilities:   s.BuildCapabilitySnapshot(),
		ReviewCount:    resolved.MarketplaceReviewCount,
		UGCCount:       resolved.UGCCount,
	}, nil
}

func (s *Service) RecordOrchestratorEvent(ctx context.Context, organizationID, runID primitive.ObjectID, traceID, phase, toolName string, status domain.AgentEventStatus, meta map[string]string) error {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return err
	}
	if run.TraceID != traceID {
		return ErrForbidden
	}
	if isTerminal(run.Status) && isTerminalPhase(phase) {
		return ErrRunTerminal
	}
	if !isAllowedOrchestratorPhase(phase) {
		return ErrInvalidInput
	}
	if err := validateEventMetadata(meta); err != nil {
		return err
	}
	return s.appendEvent(ctx, run, phase, toolName, status, meta)
}

func (s *Service) ApplyOrchestratorStatus(ctx context.Context, organizationID, runID primitive.ObjectID, traceID string, status domain.AnalysisRunStatus, currentPhase, terminalError, terminalReason string, iterationCount int) error {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return err
	}
	if run.TraceID != traceID {
		return ErrForbidden
	}
	if isTerminal(run.Status) && !isTerminal(status) {
		return ErrRunTerminal
	}
	if err := validateTransition(run.Status, status); err != nil {
		return err
	}
	run.Status = status
	if currentPhase != "" {
		run.CurrentPhase = currentPhase
	}
	if terminalError != "" {
		run.TerminalError = compliance.SanitizeForAudit(terminalError)
	}
	if terminalReason != "" {
		run.TerminalReason = terminalReason
	}
	if iterationCount > 0 {
		run.IterationCount = iterationCount
	}
	if isTerminal(status) {
		now := time.Now().UTC()
		run.CompletedAt = &now
		if s.Leases != nil {
			_ = s.Leases.Release(ctx, organizationID, runID)
		}
	}
	if status == domain.AnalysisStatusRunning && run.StartedAt == nil {
		now := time.Now().UTC()
		run.StartedAt = &now
	}
	return s.Runs.Update(ctx, organizationID, run)
}

func (s *Service) HeartbeatRunLease(ctx context.Context, organizationID, runID primitive.ObjectID, traceID, ownerID string) error {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return err
	}
	if run.TraceID != traceID {
		return ErrForbidden
	}
	if s.Leases == nil {
		return ErrGrantStoreUnavailable
	}
	if err := s.Leases.Heartbeat(ctx, organizationID, runID, ownerID); err != nil {
		return err
	}
	now := time.Now().UTC()
	run.LeaseOwnerID = ownerID
	run.LastHeartbeatAt = &now
	return s.Runs.Update(ctx, organizationID, run)
}

func (s *Service) ClaimRunLease(ctx context.Context, organizationID, runID primitive.ObjectID, traceID, ownerID string, attempt int) (bool, error) {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return false, err
	}
	if run.TraceID != traceID {
		return false, ErrForbidden
	}
	if isTerminal(run.Status) {
		return false, ErrRunTerminal
	}
	if s.Leases == nil {
		return false, ErrGrantStoreUnavailable
	}
	claimed, err := s.Leases.Claim(ctx, organizationID, runID, ownerID, attempt)
	if err != nil {
		return false, err
	}
	if claimed {
		now := time.Now().UTC()
		run.LeaseOwnerID = ownerID
		run.RecoveryAttempt = attempt
		run.LastHeartbeatAt = &now
		_ = s.Runs.Update(ctx, organizationID, run)
	}
	return claimed, nil
}

func (s *Service) IsCancellationRequested(ctx context.Context, organizationID, runID primitive.ObjectID) (bool, error) {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return false, err
	}
	return run.CancellationRequested, nil
}

func (s *Service) AuthorizeToolForOrchestrator(ctx context.Context, organizationID, runID primitive.ObjectID, traceID, toolName, toolVersion, inputHash string, stepIndex int) (*authorizeResponse, error) {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return nil, err
	}
	if run.TraceID != traceID {
		return nil, ErrForbidden
	}
	if run.CancellationRequested {
		return &authorizeResponse{Allowed: false, ErrorCode: "CANCELLED"}, nil
	}
	if !s.Registry.IsPlannable(toolName) {
		return &authorizeResponse{Allowed: false, ErrorCode: "TOOL_UNAVAILABLE"}, nil
	}
	def, ok := s.Registry.Get(toolName)
	if !ok || def.Version != toolVersion {
		return &authorizeResponse{Allowed: false, ErrorCode: "TOOL_UNAVAILABLE"}, nil
	}
	if err := ValidateToolInputHash(toolName, inputHash); err != nil {
		return &authorizeResponse{Allowed: false, ErrorCode: err.Error()}, nil
	}

	resolved, ok := s.getResolved(runID.Hex())
	if !ok {
		return nil, ErrInvalidInput
	}
	toolInput := ToolInput{
		OrganizationID: organizationID,
		RunID:          runID,
		ProductID:      run.ProductID,
		ActorID:        run.CreatedByUserID,
		Resolved:       resolved,
		Role:           domain.RoleAnalyst,
	}

	grant, err := s.Authorizer.Authorize(ctx, domain.RoleAnalyst, toolInput, toolName, toolVersion, inputHash)
	if err != nil {
		return &authorizeResponse{Allowed: false, ErrorCode: err.Error()}, nil
	}
	if s.Grants == nil {
		return &authorizeResponse{Allowed: false, ErrorCode: ErrGrantStoreUnavailable.Error()}, nil
	}
	if err := s.Grants.Issue(ctx, grant, RegistryVersion(s.Registry), traceID); err != nil {
		return &authorizeResponse{Allowed: false, ErrorCode: err.Error()}, nil
	}

	return &authorizeResponse{
		Allowed:         true,
		GrantNonce:      grant.Nonce,
		ToolExecutionID: grant.ToolExecutionID.Hex(),
	}, nil
}

func (s *Service) ExecuteToolForOrchestrator(ctx context.Context, organizationID, runID primitive.ObjectID, traceID, toolName, toolVersion, inputHash, grantNonce, toolExecutionIDHex string, stepIndex int) (*executeResponse, error) {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return nil, err
	}
	if run.TraceID != traceID {
		return nil, ErrForbidden
	}
	if run.CancellationRequested {
		return &executeResponse{Status: "REJECTED", Error: "cancelled"}, nil
	}
	if s.Grants == nil {
		return &executeResponse{Status: "REJECTED", Error: ErrGrantStoreUnavailable.Error()}, nil
	}

	binding := GrantBinding{
		OrganizationID: organizationID,
		AnalysisRunID:  runID,
		UserID:         run.CreatedByUserID,
		ToolName:       toolName,
		ToolVersion:    toolVersion,
		InputHash:      inputHash,
		TraceID:        traceID,
	}
	record, err := s.Grants.Consume(ctx, grantNonce, binding)
	if err != nil {
		return &executeResponse{Status: "REJECTED", Error: err.Error()}, nil
	}

	resolved, ok := s.getResolved(runID.Hex())
	if !ok {
		return nil, ErrInvalidInput
	}
	toolInput := ToolInput{
		OrganizationID: organizationID,
		RunID:          runID,
		ProductID:      run.ProductID,
		ActorID:        run.CreatedByUserID,
		Resolved:       resolved,
		Role:           domain.RoleAnalyst,
	}

	execCtx, cancel := context.WithTimeout(ctx, perToolTimeout(toolName))
	defer cancel()

	started := time.Now().UTC()
	execID, _ := primitive.ObjectIDFromHex(toolExecutionIDHex)
	exec := &domain.ToolExecution{
		ID:                    execID,
		OrganizationID:        organizationID,
		AnalysisRunID:         runID,
		ToolName:              toolName,
		ToolVersion:           toolVersion,
		Attempt:               1,
		AuthorizationDecision: "ALLOW",
		InputHash:             inputHash,
		GrantNonceHash:        record.GrantIDHash,
		Status:                domain.ToolExecRunning,
		StartedAt:             &started,
	}
	_ = s.Executions.Create(ctx, exec)

	output, err := s.Executor.Execute(execCtx, toolName, toolInput)
	completed := time.Now().UTC()
	exec.CompletedAt = &completed
	if err != nil {
		exec.Status = domain.ToolExecFailed
		exec.ErrorCode = err.Error()
		exec.ErrorClass = classifyExecError(err)
		_ = s.Executions.Update(ctx, organizationID, exec)
		return &executeResponse{Status: "FAILED", Error: err.Error()}, nil
	}

	if err := ValidateToolOutput(toolName, output.Status, output.Payload); err != nil {
		exec.Status = domain.ToolExecFailed
		exec.ErrorCode = err.Error()
		exec.ErrorClass = "SCHEMA"
		_ = s.Executions.Update(ctx, organizationID, exec)
		return &executeResponse{Status: "FAILED", Error: err.Error()}, nil
	}

	if err := s.validateToolOutputCompliance(ctx, run, toolInput, output); err != nil {
		exec.Status = domain.ToolExecFailed
		exec.ErrorCode = err.Error()
		exec.ErrorClass = "COMPLIANCE"
		_ = s.Executions.Update(ctx, organizationID, exec)
		return &executeResponse{Status: "FAILED", Error: err.Error()}, nil
	}

	exec.Status = domain.ToolExecCompleted
	exec.OutputHash = HashPayload(output.Payload)
	_ = s.Executions.Update(ctx, organizationID, exec)
	return &executeResponse{Status: output.Status, Payload: output.Payload}, nil
}

func (s *Service) validateToolOutputCompliance(ctx context.Context, run *domain.AnalysisRun, input ToolInput, output *ToolOutput) error {
	if s.Compliance == nil {
		return nil
	}
	raw, _ := json.Marshal(output.Payload)
	result, err := s.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "analysis_tool_output",
		OrganizationID: input.OrganizationID,
		UserID:         input.ActorID,
		Purpose:        domain.ConsentPurposeDataProcessing,
		OutputText:     string(raw),
	})
	if err != nil {
		return err
	}
	if !result.Allowed {
		return ErrComplianceRejected
	}
	return nil
}

func (s *Service) appendEvent(ctx context.Context, run *domain.AnalysisRun, phase, tool string, status domain.AgentEventStatus, meta map[string]string) error {
	seq, err := s.Events.NextSequence(ctx, run.OrganizationID, run.ID)
	if err != nil {
		return err
	}
	if meta == nil {
		meta = map[string]string{}
	}
	meta["traceId"] = run.TraceID
	return s.Events.Append(ctx, &domain.AgentRunEvent{
		OrganizationID: run.OrganizationID,
		RunID:          run.ID,
		Sequence:       seq,
		Phase:          phase,
		ToolName:       tool,
		Status:         status,
		Metadata:       meta,
		TraceID:        run.TraceID,
	})
}

func validateEventMetadata(meta map[string]string) error {
	if meta == nil {
		return nil
	}
	total := 0
	for k, v := range meta {
		total += len(k) + len(v)
		if strings.Contains(strings.ToLower(v), "password") || strings.Contains(strings.ToLower(v), "token") {
			return fmt.Errorf("%w: sensitive metadata rejected", ErrInvalidInput)
		}
	}
	if total > maxEventMetadataBytes {
		return fmt.Errorf("%w: metadata too large", ErrInvalidInput)
	}
	return nil
}

func isTerminalPhase(phase string) bool {
	switch phase {
	case domain.AgentPhaseRunCompleted, domain.AgentPhaseRunFailed, domain.AgentPhaseRunCancelled:
		return true
	default:
		return false
	}
}

func perToolTimeout(toolName string) time.Duration {
	switch toolName {
	case "review_sampler", "dataset_validator":
		return 30 * time.Second
	default:
		return 15 * time.Second
	}
}

func classifyExecError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "compliance"):
		return "COMPLIANCE"
	case strings.Contains(msg, "schema"):
		return "SCHEMA"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "TIMEOUT"
	default:
		return "RUNTIME"
	}
}

func validateTransition(from, to domain.AnalysisRunStatus) error {
	if from == to {
		return nil
	}
	if isTerminal(from) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	allowed := map[domain.AnalysisRunStatus][]domain.AnalysisRunStatus{
		domain.AnalysisStatusPending:   {domain.AnalysisStatusRunning, domain.AnalysisStatusFailed, domain.AnalysisStatusRejected},
		domain.AnalysisStatusRunning:   {domain.AnalysisStatusCompleted, domain.AnalysisStatusFailed, domain.AnalysisStatusRejected},
		domain.AnalysisStatusCompleted: {},
		domain.AnalysisStatusFailed:    {},
		domain.AnalysisStatusRejected:  {},
	}
	for _, next := range allowed[from] {
		if next == to {
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}

func isTerminal(status domain.AnalysisRunStatus) bool {
	switch status {
	case domain.AnalysisStatusCompleted, domain.AnalysisStatusFailed, domain.AnalysisStatusRejected:
		return true
	default:
		return false
	}
}

func isAllowedOrchestratorPhase(phase string) bool {
	switch phase {
	case domain.AgentPhaseRunStarted, domain.AgentPhaseCompliancePrecheck, domain.AgentPhasePlanCreated,
		domain.AgentPhaseToolSelected, domain.AgentPhaseToolExecutionStarted, domain.AgentPhaseToolExecutionCompleted,
		domain.AgentPhaseObservationCreated, domain.AgentPhaseRunCompleted, domain.AgentPhaseRunFailed,
		domain.AgentPhaseRunCancelled:
		return true
	default:
		return false
	}
}

var errNotFound = repository.ErrNotFound
