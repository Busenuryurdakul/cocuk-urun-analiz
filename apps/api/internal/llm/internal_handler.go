package llm

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const internalTokenHeader = "X-Miyuna-Internal-Token"

type InternalHandler struct {
	Gateway *Gateway
	Token   string
}

func (h *InternalHandler) Register(r chi.Router) {
	r.Route("/internal/llm/v1", func(r chi.Router) {
		r.Use(h.middleware)
		r.Post("/complete", h.handleComplete)
		r.Get("/health", h.handleHealth)
	})
}

func (h *InternalHandler) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Token == "" || r.Header.Get(internalTokenHeader) != h.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type completeRequest struct {
	OrganizationID       string   `json:"organizationId"`
	UserID               string   `json:"userId,omitempty"`
	AnalysisRunID        string   `json:"analysisRunId,omitempty"`
	CorrelationID        string   `json:"correlationId"`
	IdempotencyKey       string   `json:"idempotencyKey,omitempty"`
	TaskType             string   `json:"taskType"`
	SafetyRisk           string   `json:"safetyRisk,omitempty"`
	RequireEvidence      bool     `json:"requireEvidence,omitempty"`
	PersonaKey           string   `json:"personaKey,omitempty"`
	UserPrompt           string   `json:"userPrompt"`
	ExternalContent      []string `json:"externalContent,omitempty"`
	RoutingPolicyVersion string   `json:"routingPolicyVersion,omitempty"`
	ConfigSnapshotID     string   `json:"configSnapshotId,omitempty"`
}

func (h *InternalHandler) handleComplete(w http.ResponseWriter, r *http.Request) {
	var req completeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.OrganizationID) == "" {
		http.Error(w, "organizationId required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UserPrompt) == "" {
		http.Error(w, "userPrompt required", http.StatusBadRequest)
		return
	}
	orgID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.OrganizationID))
	if err != nil {
		http.Error(w, "invalid organizationId", http.StatusBadRequest)
		return
	}
	gwReq := GatewayRequest{
		OrganizationID:       orgID,
		CorrelationID:        strings.TrimSpace(req.CorrelationID),
		IdempotencyKey:       strings.TrimSpace(req.IdempotencyKey),
		TaskType:             strings.TrimSpace(req.TaskType),
		SafetyRisk:           strings.TrimSpace(req.SafetyRisk),
		RequireEvidence:      req.RequireEvidence,
		PersonaKey:           strings.TrimSpace(req.PersonaKey),
		UserPrompt:           req.UserPrompt,
		ExternalContent:      req.ExternalContent,
		RoutingPolicyVersion: strings.TrimSpace(req.RoutingPolicyVersion),
	}
	if gwReq.TaskType == "" {
		gwReq.TaskType = "analysis"
	}
	if gwReq.CorrelationID == "" {
		gwReq.CorrelationID = primitive.NewObjectID().Hex()
	}
	if req.UserID != "" {
		if uid, err := primitive.ObjectIDFromHex(req.UserID); err == nil {
			gwReq.UserID = &uid
		}
	}
	if req.AnalysisRunID != "" {
		if rid, err := primitive.ObjectIDFromHex(req.AnalysisRunID); err == nil {
			gwReq.AnalysisRunID = &rid
		}
	}
	if req.ConfigSnapshotID != "" {
		if sid, err := primitive.ObjectIDFromHex(req.ConfigSnapshotID); err == nil {
			gwReq.ConfigSnapshotID = &sid
		}
	}
	resp, err := h.Gateway.Complete(r.Context(), gwReq)
	if err != nil {
		writeLLMError(w, err)
		return
	}
	writeJSON(w, resp)
}

func (h *InternalHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	items, err := h.Gateway.Health(r.Context())
	if err != nil {
		writeLLMError(w, err)
		return
	}
	writeJSON(w, map[string]any{"models": items})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeLLMError(w http.ResponseWriter, err error) {
	switch {
	case err == ErrForbidden, err == ErrComplianceBlocked, err == ErrPromptInjection:
		http.Error(w, err.Error(), http.StatusForbidden)
	case err == ErrInvalidInput, err == ErrInvalidPolicyVersion, err == ErrPublishValidation, err == ErrDraftNotValidated:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
