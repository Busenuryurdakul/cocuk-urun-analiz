//go:build real_llm

package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/analysis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/llm"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRealLLMFinalPipelineE2E(t *testing.T) {
	if strings.TrimSpace(os.Getenv("LLM_USE_MOCK")) != "false" {
		t.Skip("set LLM_USE_MOCK=false for real_llm pipeline E2E")
	}
	primary := strings.TrimSpace(os.Getenv("LLM_PRIMARY_MODEL_NAME"))
	secondary := strings.TrimSpace(os.Getenv("LLM_SECONDARY_MODEL_NAME"))
	if primary == "" || secondary == "" || primary == secondary {
		t.Skip("distinct LLM_PRIMARY_MODEL_NAME and LLM_SECONDARY_MODEL_NAME required")
	}

	h := NewPhase5Harness(t, WithLLMUseMock(false))

	runID := seedAnalysisRun(t, h)
	rating := 1.0
	_, err := h.App.Agent.Executor.Execute(h.Ctx, "review_analyzer", agent.ToolInput{
		OrganizationID: h.OrgID, RunID: runID, ProductID: h.ProductID, Role: domain.RoleAnalyst,
		Resolved: agent.ResolvedInput{MarketplaceReviews: []domain.MarketplaceReview{{
			ID: primitive.NewObjectID(), OrganizationID: h.OrgID, ProductID: h.ProductID,
			ReviewText: "small parts choking hazard", Rating: &rating,
		}}},
	})
	if err != nil {
		t.Fatalf("review_analyzer: %v", err)
	}

	ev, err := h.App.Evidence.CreateEvidence(h.Ctx, evidence.CreateEvidenceInput{
		OrganizationID: h.OrgID, AnalysisRunID: runID, ProductID: h.ProductID, ClaimID: "real-e2e",
		Source: "https://www.cpsc.gov/Recalls/123", SourceType: "OFFICIAL_RECALL",
		Claim: "Official recall warning", Snippet: "choking", Reference: "https://www.cpsc.gov/Recalls/123",
		Reliability: 0.95, Freshness: 0.9,
	})
	if err != nil {
		t.Fatalf("evidence: %v", err)
	}

	h.App.Safety.Adapters = []safety.SourceAdapter{fixtureRecallAdapter{records: []recall.Record{{
		Source: recall.SourceCPSC, SourceRecordID: "123", Title: "Recall", Reference: "https://www.cpsc.gov/Recalls/123",
	}}}}
	_, err = h.App.Agent.Executor.Execute(h.Ctx, "safety_analyzer", agent.ToolInput{
		OrganizationID: h.OrgID, RunID: runID, ProductID: h.ProductID, Role: domain.RoleAnalyst,
		EvidenceIDs: []string{ev.ID.Hex()},
	})
	if err != nil {
		t.Fatalf("safety_analyzer: %v", err)
	}

	userID := h.AnalystID
	workerGW, err := h.App.LLM.Gateway.Complete(h.Ctx, llm.GatewayRequest{
		OrganizationID: h.OrgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "real-pipeline-worker",
		IdempotencyKey: "real-pipeline-worker-" + runID.Hex(),
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Summarize safety evidence for a children's product with an official recall. Be concise.",
	})
	if err != nil {
		t.Fatalf("real worker gateway: %v", err)
	}

	reviewerGW, err := h.App.LLM.Gateway.Complete(h.Ctx, llm.GatewayRequest{
		OrganizationID: h.OrgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "real-pipeline-reviewer",
		IdempotencyKey: "real-pipeline-reviewer-" + runID.Hex(),
		TaskType:       "review",
		PersonaKey:     domain.PersonaResultAnalyst,
		UserPrompt:     "Review the worker summary for unsupported claims. Be concise.",
	})
	if err != nil {
		t.Fatalf("real reviewer gateway: %v", err)
	}

	final, err := h.App.Analysis.Finalize(h.Ctx, analysis.FinalizeInput{
		OrganizationID: h.OrgID,
		AnalysisRunID:  runID,
		Worker: domain.AnalysisLLMResult{
			Provider: workerGW.ProviderKey,
			Model:    workerGW.ModelKey,
			Persona:  workerGW.PersonaKey,
			Output:   workerGW.Content,
		},
		Reviewer: domain.AnalysisLLMResult{
			Provider: reviewerGW.ProviderKey,
			Model:    reviewerGW.ModelKey,
			Persona:  reviewerGW.PersonaKey,
			Output:   reviewerGW.Content,
		},
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if final.SchemaVersion != domain.FinalAnalysisSchemaVersion {
		t.Fatalf("schema=%s", final.SchemaVersion)
	}
	if final.Decision == "" {
		t.Fatal("expected deterministic decision")
	}

	token := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(token, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			finalResult { schemaVersion decision confidence summary }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("graphql: %v", errs)
	}
	run := data["agentRun"].(map[string]any)
	if run["finalResult"] == nil {
		t.Fatal("expected finalResult via GraphQL")
	}
}
