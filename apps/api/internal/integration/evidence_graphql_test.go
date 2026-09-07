package integration

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGraphQLAnalysisEvidenceReadAndIDOR(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)

	created, err := h.App.Evidence.CreateEvidence(h.Ctx, evidence.CreateEvidenceInput{
		OrganizationID: h.OrgID,
		AnalysisRunID:  runID,
		ProductID:      h.ProductID,
		ClaimID:        "claim-123",
		Source:         "https://example.test/review",
		SourceType:     "URL",
		Claim:          "Ürün boğulma riski taşıyor.",
		Snippet:        "small parts",
		Reference:      "https://example.test/review",
		Reliability:    0.85,
		Freshness:      0.75,
	})
	if err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}

	tokenA := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(tokenA, `query($orgId: ID!, $runId: ID!) {
		analysisEvidence(organizationId: $orgId, analysisRunId: $runId) {
			id organizationId analysisRunId claimId source reliability
		}
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			id
			evidence { id organizationId }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("org A read failed: %v", errs)
	}
	items := data["analysisEvidence"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(items))
	}
	gotID := items[0].(map[string]any)["id"].(string)
	if gotID != created.ID.Hex() {
		t.Fatalf("expected evidence %s, got %s", created.ID.Hex(), gotID)
	}
	run := data["agentRun"].(map[string]any)
	nested := run["evidence"].([]any)
	if len(nested) != 1 {
		t.Fatalf("expected 1 nested evidence, got %d", len(nested))
	}

	tokenB := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errsB, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisEvidence(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsB, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for org B token + org A run, got %v", errsB)
	}

	_, errsRun, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) { evidence { id } }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsRun, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN nested evidence via agentRun, got %v", errsRun)
	}

	dataB, errsOwn, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisEvidence(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OtherOrgID.Hex(), "runId": runID.Hex()})
	if len(errsOwn) > 0 {
		t.Fatalf("org B querying own org + foreign run should not leak: %v", errsOwn)
	}
	if leaked := dataB["analysisEvidence"].([]any); len(leaked) != 0 {
		t.Fatalf("foreign run id must not return org A evidence, got %d", len(leaked))
	}

	_, errsForge, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisEvidence(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OtherOrgID.Hex(), "runId": primitive.NewObjectID().Hex()})
	if len(errsForge) > 0 {
		t.Fatalf("unknown run in own org should be empty, not error: %v", errsForge)
	}
}
