package integration

import (
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/analysis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCorePipelineMockFinalResultAndTenantIsolation(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	if err := h.App.Products.Mappings.Create(h.Ctx, &domain.ProductSourceMapping{
		OrganizationID:  h.OrgID,
		ProductID:       h.ProductID,
		Source:          domain.MarketplaceAmazon,
		SourceProductID: "final-fixture-1",
		Model:           "X1",
		Brand:           "Brand",
		UPC:             "012345678905",
	}); err != nil {
		t.Fatal(err)
	}

	rating := 1.0
	reviewOut, err := h.App.Agent.Executor.Execute(h.Ctx, "review_analyzer", agent.ToolInput{
		OrganizationID: h.OrgID, RunID: runID, ProductID: h.ProductID, Role: domain.RoleAnalyst,
		Resolved: agent.ResolvedInput{MarketplaceReviews: []domain.MarketplaceReview{{
			ID: primitive.NewObjectID(), OrganizationID: h.OrgID, ProductID: h.ProductID,
			ReviewText: "choking hazard small parts, broke after one day", Rating: &rating,
		}}},
	})
	if err != nil {
		t.Fatalf("review_analyzer: %v", err)
	}
	if reviewOut.Status != "OK" {
		t.Fatalf("review status=%s", reviewOut.Status)
	}

	ev, err := h.App.Evidence.CreateEvidence(h.Ctx, evidence.CreateEvidenceInput{
		OrganizationID: h.OrgID, AnalysisRunID: runID, ProductID: h.ProductID, ClaimID: "analysis-primary",
		Source: "https://www.cpsc.gov/Recalls/123", SourceType: "OFFICIAL_RECALL",
		Claim: "Official recall warning for this product", Snippet: "choking",
		Reference: "https://www.cpsc.gov/Recalls/123", Reliability: 0.95, Freshness: 0.9,
	})
	if err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}

	valOut, err := h.App.Agent.Executor.Execute(h.Ctx, "evidence_validator", agent.ToolInput{
		OrganizationID: h.OrgID, RunID: runID, ProductID: h.ProductID, Role: domain.RoleAnalyst,
		ClaimID: "analysis-primary", ClaimText: "Official recall warning for this product",
		EvidenceIDs: []string{ev.ID.Hex()},
	})
	if err != nil {
		t.Fatalf("evidence_validator: %v", err)
	}
	if valOut.Payload["status"] == "UNSUPPORTED" {
		t.Fatalf("official evidence should not be UNSUPPORTED: %v", valOut.Payload)
	}

	h.App.Safety.Adapters = []safety.SourceAdapter{fixtureRecallAdapter{records: []recall.Record{{
		Source: recall.SourceCPSC, SourceRecordID: "123", Title: "Brand Test Product Recall",
		Brand: "Brand", ProductName: "Test Product", Model: "X1", Manufacturer: "Brand",
		UPC: "012345678905", Hazard: "Choking", Reference: "https://www.cpsc.gov/Recalls/123",
	}}}}
	safetyOut, err := h.App.Agent.Executor.Execute(h.Ctx, "safety_analyzer", agent.ToolInput{
		OrganizationID: h.OrgID, RunID: runID, ProductID: h.ProductID, Role: domain.RoleAnalyst,
		EvidenceIDs: []string{ev.ID.Hex()},
	})
	if err != nil {
		t.Fatalf("safety_analyzer: %v", err)
	}
	if safetyOut.Status != "OK" {
		t.Fatalf("safety status=%s", safetyOut.Status)
	}

	final, err := h.App.Analysis.Finalize(h.Ctx, analysis.FinalizeInput{
		OrganizationID: h.OrgID, AnalysisRunID: runID,
		Worker:   domain.AnalysisLLMResult{Provider: "mock", Model: "careful_analyst", Persona: "careful_analyst", Output: "Confirmed official recall. Block this product."},
		Reviewer: domain.AnalysisLLMResult{Provider: "mock", Model: "result_analyst", Persona: "result_analyst", Output: "Worker recall is supported by official evidence."},
	})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if final.Decision != domain.DecisionBlock {
		t.Fatalf("expected BLOCK, got %s", final.Decision)
	}
	if final.SchemaVersion != domain.FinalAnalysisSchemaVersion {
		t.Fatalf("schema=%s", final.SchemaVersion)
	}

	tokenA := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(tokenA, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			id
			reviewInsights { topic count kind }
			finalResult { schemaVersion summary overallRisk confidence decision recommendation limitations hallucinationFlags createdAt }
			evidence { source claim supportStatus reference }
			safetyFindings { severity type }
			recalls { source matched requiresReview }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("org A read failed: %v", errs)
	}
	run := data["agentRun"].(map[string]any)
	fr := run["finalResult"].(map[string]any)
	if fr["decision"] != "BLOCK" {
		t.Fatalf("graphql decision=%v", fr["decision"])
	}
	if run["reviewInsights"] == nil {
		t.Fatal("expected reviewInsights")
	}

	tokenB := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errsB, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			finalResult { decision }
			reviewInsights { topic }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsB, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for org B reading org A finalResult, got %v", errsB)
	}

	dataB, errsOwn, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			finalResult { decision }
		}
	}`, map[string]any{"orgId": h.OtherOrgID.Hex(), "runId": runID.Hex()})
	if len(errsOwn) == 0 {
		if runB, ok := dataB["agentRun"].(map[string]any); ok && runB != nil && runB["finalResult"] != nil {
			t.Fatalf("foreign run must not leak org A finalResult: %v", runB["finalResult"])
		}
	} else if !errHasCode(errsOwn, "NOT_FOUND") && !errHasCode(errsOwn, "FORBIDDEN") && !errHasMessage(errsOwn, "not found") {
		t.Fatalf("org B own-org + foreign run should not leak: %v", errsOwn)
	}
}

func TestUnauthorizedAnalysisReadAndForeignProduct(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	_, errs, _ := h.GraphQL("", `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) { finalResult { decision } }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errs, "UNAUTHORIZED") && !errHasCode(errs, "UNAUTHENTICATED") && len(errs) == 0 {
		t.Fatalf("expected unauthorized analysis read, got %v", errs)
	}

	tokenB := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errsB, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			evidence { id }
			safetyFindings { id }
			finalResult { decision }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsB, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN cross-org analysis read, got %v", errsB)
	}
}

func errHasMessage(errs []map[string]any, needle string) bool {
	for _, e := range errs {
		msg, _ := e["message"].(string)
		if strings.Contains(strings.ToLower(msg), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
