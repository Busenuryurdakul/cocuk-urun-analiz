package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InternalHandler serves Go authoritative boundaries for the Python orchestrator.
type InternalHandler struct {
	Service *Service
	Token   string
}

func (h *InternalHandler) Register(r chi.Router) {
	r.Route("/internal/agent/v1", func(r chi.Router) {
		r.Use(h.Middleware)
		r.Post("/events", h.handleEvent)
		r.Post("/runs/status", h.handleRunStatus)
		r.Get("/runs/context", h.handleRunContext)
		r.Get("/capabilities", h.handleCapabilities)
		r.Post("/tools/authorize", h.handleAuthorize)
		r.Post("/tools/execute", h.handleExecute)
		r.Post("/runs/finalize", h.handleFinalize)
		r.Get("/runs/cancellation", h.handleCancellation)
		r.Post("/runs/heartbeat", h.handleHeartbeat)
		r.Post("/runs/lease/claim", h.handleLeaseClaim)
	})
}

func (h *InternalHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Token == "" || r.Header.Get(internalTokenHeader) != h.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type recordEventRequest struct {
	OrganizationID string            `json:"organizationId"`
	AnalysisRunID  string            `json:"analysisRunId"`
	TraceID        string            `json:"traceId"`
	Phase          string            `json:"phase"`
	ToolName       string            `json:"toolName,omitempty"`
	Status         string            `json:"status"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (h *InternalHandler) handleEvent(w http.ResponseWriter, r *http.Request) {
	var req recordEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	if err := h.Service.RecordOrchestratorEvent(r.Context(), orgID, runID, req.TraceID, req.Phase, req.ToolName, domain.AgentEventStatus(req.Status), req.Metadata); err != nil {
		writeAgentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type runStatusRequest struct {
	OrganizationID string `json:"organizationId"`
	AnalysisRunID  string `json:"analysisRunId"`
	TraceID        string `json:"traceId"`
	Status         string `json:"status"`
	CurrentPhase   string `json:"currentPhase,omitempty"`
	TerminalError  string `json:"terminalError,omitempty"`
	TerminalReason string `json:"terminalReason,omitempty"`
	IterationCount int    `json:"iterationCount,omitempty"`
}

func (h *InternalHandler) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	var req runStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	if err := h.Service.ApplyOrchestratorStatus(r.Context(), orgID, runID, req.TraceID, domain.AnalysisRunStatus(req.Status), req.CurrentPhase, req.TerminalError, req.TerminalReason, req.IterationCount); err != nil {
		writeAgentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InternalHandler) handleRunContext(w http.ResponseWriter, r *http.Request) {
	orgID, runID, err := parseOrgRun(r.URL.Query().Get("organizationId"), r.URL.Query().Get("analysisRunId"))
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	ctx, err := h.Service.RunContext(r.Context(), orgID, runID)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, ctx)
}

func (h *InternalHandler) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.Service.BuildCapabilitySnapshot())
}

type authorizeRequest struct {
	OrganizationID string `json:"organizationId"`
	AnalysisRunID  string `json:"analysisRunId"`
	TraceID        string `json:"traceId"`
	ToolName       string `json:"toolName"`
	ToolVersion    string `json:"toolVersion"`
	InputHash      string `json:"inputHash"`
	StepIndex      int    `json:"stepIndex"`
}

type authorizeResponse struct {
	Allowed         bool   `json:"allowed"`
	GrantNonce      string `json:"grantNonce,omitempty"`
	ToolExecutionID string `json:"toolExecutionId,omitempty"`
	ErrorCode       string `json:"errorCode,omitempty"`
}

func (h *InternalHandler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	var req authorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	resp, err := h.Service.AuthorizeToolForOrchestrator(r.Context(), orgID, runID, req.TraceID, req.ToolName, req.ToolVersion, req.InputHash, req.StepIndex)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, resp)
}

type executeRequest struct {
	OrganizationID  string   `json:"organizationId"`
	AnalysisRunID   string   `json:"analysisRunId"`
	TraceID         string   `json:"traceId"`
	ToolName        string   `json:"toolName"`
	ToolVersion     string   `json:"toolVersion"`
	InputHash       string   `json:"inputHash"`
	GrantNonce      string   `json:"grantNonce"`
	ToolExecutionID string   `json:"toolExecutionId"`
	StepIndex       int      `json:"stepIndex"`
	ClaimID         string   `json:"claimId,omitempty"`
	ClaimText       string   `json:"claimText,omitempty"`
	EvidenceIDs     []string `json:"evidenceIds,omitempty"`
	ReviewIDs       []string `json:"reviewIds,omitempty"`
	RecallIDs       []string `json:"recallIds,omitempty"`
}

type executeResponse struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload,omitempty"`
	Error   string         `json:"error,omitempty"`
}

func (h *InternalHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	resp, err := h.Service.ExecuteToolForOrchestrator(r.Context(), orgID, runID, req.TraceID, req.ToolName, req.ToolVersion, req.InputHash, req.GrantNonce, req.ToolExecutionID, req.StepIndex, orchestratorExecuteExtras{
		ClaimID:     req.ClaimID,
		ClaimText:   req.ClaimText,
		EvidenceIDs: req.EvidenceIDs,
		ReviewIDs:   req.ReviewIDs,
		RecallIDs:   req.RecallIDs,
	})
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, resp)
}

type llmResultPayload struct {
	Provider             string   `json:"provider"`
	Model                string   `json:"model"`
	Persona              string   `json:"persona"`
	RoutingPolicyVersion string   `json:"routingPolicyVersion,omitempty"`
	ConfigSnapshotID     string   `json:"configSnapshotId,omitempty"`
	Output               string   `json:"output"`
	Flags                []string `json:"flags,omitempty"`
}

type finalizeRequest struct {
	OrganizationID string           `json:"organizationId"`
	AnalysisRunID  string           `json:"analysisRunId"`
	TraceID        string           `json:"traceId"`
	ComplianceMax  string           `json:"complianceMax,omitempty"`
	Worker         llmResultPayload `json:"worker"`
	Reviewer       llmResultPayload `json:"reviewer"`
}

func (h *InternalHandler) handleFinalize(w http.ResponseWriter, r *http.Request) {
	var req finalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	run, err := h.Service.Runs.FindByID(r.Context(), orgID, runID)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	if run.TraceID != req.TraceID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	final, err := h.Service.FinalizeForOrchestrator(r.Context(), orgID, runID, domain.AnalysisLLMResult{
		Provider:             req.Worker.Provider,
		Model:                req.Worker.Model,
		Persona:              req.Worker.Persona,
		RoutingPolicyVersion: req.Worker.RoutingPolicyVersion,
		ConfigSnapshotID:     req.Worker.ConfigSnapshotID,
		Output:               req.Worker.Output,
	}, domain.AnalysisLLMResult{
		Provider:             req.Reviewer.Provider,
		Model:                req.Reviewer.Model,
		Persona:              req.Reviewer.Persona,
		RoutingPolicyVersion: req.Reviewer.RoutingPolicyVersion,
		ConfigSnapshotID:     req.Reviewer.ConfigSnapshotID,
		Output:               req.Reviewer.Output,
		Flags:                req.Reviewer.Flags,
	}, req.ComplianceMax)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"schemaVersion": final.SchemaVersion,
		"decision":      final.Decision,
		"confidence":    final.Confidence,
		"overallRisk":   final.OverallRisk,
	})
}

func (h *InternalHandler) handleCancellation(w http.ResponseWriter, r *http.Request) {
	orgID, runID, err := parseOrgRun(r.URL.Query().Get("organizationId"), r.URL.Query().Get("analysisRunId"))
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	cancelled, err := h.Service.IsCancellationRequested(r.Context(), orgID, runID)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, map[string]bool{"cancellationRequested": cancelled})
}

type heartbeatRequest struct {
	OrganizationID string `json:"organizationId"`
	AnalysisRunID  string `json:"analysisRunId"`
	TraceID        string `json:"traceId"`
	OwnerID        string `json:"ownerId"`
}

func (h *InternalHandler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	if err := h.Service.HeartbeatRunLease(r.Context(), orgID, runID, req.TraceID, req.OwnerID); err != nil {
		writeAgentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type leaseClaimRequest struct {
	OrganizationID string `json:"organizationId"`
	AnalysisRunID  string `json:"analysisRunId"`
	TraceID        string `json:"traceId"`
	OwnerID        string `json:"ownerId"`
	Attempt        int    `json:"attempt"`
}

func (h *InternalHandler) handleLeaseClaim(w http.ResponseWriter, r *http.Request) {
	var req leaseClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	orgID, runID, err := parseOrgRun(req.OrganizationID, req.AnalysisRunID)
	if err != nil {
		http.Error(w, "invalid ids", http.StatusBadRequest)
		return
	}
	claimed, err := h.Service.ClaimRunLease(r.Context(), orgID, runID, req.TraceID, req.OwnerID, req.Attempt)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	writeJSON(w, map[string]bool{"claimed": claimed})
}

func parseOrgRun(orgHex, runHex string) (primitive.ObjectID, primitive.ObjectID, error) {
	orgID, err := primitive.ObjectIDFromHex(strings.TrimSpace(orgHex))
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, err
	}
	runID, err := primitive.ObjectIDFromHex(strings.TrimSpace(runHex))
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, err
	}
	return orgID, runID, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeAgentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrAuthorizationDenied), errors.Is(err, ErrToolUnavailable):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidTransition):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, repository.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
