package integration

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fixtureRecallAdapter struct {
	records []recall.Record
}

func (f fixtureRecallAdapter) Source() string { return recall.SourceCPSC }
func (f fixtureRecallAdapter) SearchRecalls(_ context.Context, _ recall.Identity) ([]recall.Record, error) {
	return f.records, nil
}

func TestGraphQLSafetyFindingsRecallsAndIDOR(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	if err := h.App.Products.Mappings.Create(h.Ctx, &domain.ProductSourceMapping{
		OrganizationID:  h.OrgID,
		ProductID:       h.ProductID,
		Source:          domain.MarketplaceAmazon,
		SourceProductID: "safety-fixture-1",
		Model:           "X1",
		Brand:           "Brand",
		UPC:             "012345678905",
	}); err != nil {
		t.Fatal(err)
	}
	h.App.Safety.Adapters = []safety.SourceAdapter{fixtureRecallAdapter{records: []recall.Record{{
		Source: recall.SourceCPSC, SourceRecordID: "123", Title: "Brand Test Product Recall",
		Brand: "Brand", ProductName: "Test Product", Model: "X1", Manufacturer: "Brand",
		UPC: "012345678905", Hazard: "Choking", Reference: "https://www.cpsc.gov/Recalls/123",
	}}}}

	out, err := h.App.Agent.Executor.Execute(h.Ctx, "safety_analyzer", agent.ToolInput{
		OrganizationID: h.OrgID,
		RunID:          runID,
		ProductID:      h.ProductID,
		Role:           domain.RoleAnalyst,
		EvidenceIDs:    []string{},
	})
	if err != nil {
		t.Fatalf("Execute safety_analyzer: %v", err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	if len(findings) != 1 || findings[0]["severity"] != "CRITICAL" {
		t.Fatalf("expected confirmed critical finding, got %v", findings)
	}

	tokenA := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(tokenA, `query($orgId: ID!, $runId: ID!) {
		analysisSafetyFindings(organizationId: $orgId, analysisRunId: $runId) {
			id type severity confidence evidenceIds rationale
		}
		analysisRecallMatches(organizationId: $orgId, analysisRunId: $runId) {
			source sourceRecordId matched confidence method requiresReview
		}
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			id
			safetyFindings { id type severity }
			recalls { source sourceRecordId matched requiresReview }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("org A read failed: %v", errs)
	}
	if got := data["analysisSafetyFindings"].([]any); len(got) != 1 {
		t.Fatalf("expected 1 safety finding, got %d", len(got))
	}
	if got := data["analysisRecallMatches"].([]any); len(got) != 1 {
		t.Fatalf("expected 1 recall match, got %d", len(got))
	}
	run := data["agentRun"].(map[string]any)
	if nested := run["safetyFindings"].([]any); len(nested) != 1 {
		t.Fatalf("expected nested safety findings, got %d", len(nested))
	}
	if nested := run["recalls"].([]any); len(nested) != 1 {
		t.Fatalf("expected nested recalls, got %d", len(nested))
	}

	tokenB := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errsB, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisSafetyFindings(organizationId: $orgId, analysisRunId: $runId) { id }
		analysisRecallMatches(organizationId: $orgId, analysisRunId: $runId) { source }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsB, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for org B token + org A run, got %v", errsB)
	}

	_, errsRun, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) {
			safetyFindings { id }
			recalls { source }
		}
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errsRun, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN nested safety via agentRun, got %v", errsRun)
	}

	dataB, errsOwn, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisSafetyFindings(organizationId: $orgId, analysisRunId: $runId) { id }
		analysisRecallMatches(organizationId: $orgId, analysisRunId: $runId) { source }
	}`, map[string]any{"orgId": h.OtherOrgID.Hex(), "runId": runID.Hex()})
	if len(errsOwn) > 0 {
		t.Fatalf("org B querying own org + foreign run should not leak: %v", errsOwn)
	}
	if leaked := dataB["analysisSafetyFindings"].([]any); len(leaked) != 0 {
		t.Fatalf("foreign run id must not return org A findings, got %d", len(leaked))
	}
	if leaked := dataB["analysisRecallMatches"].([]any); len(leaked) != 0 {
		t.Fatalf("foreign run id must not return org A recall matches, got %d", len(leaked))
	}

	_, errsForge, _ := h.GraphQL(tokenB, `query($orgId: ID!, $runId: ID!) {
		analysisSafetyFindings(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OtherOrgID.Hex(), "runId": primitive.NewObjectID().Hex()})
	if len(errsForge) > 0 {
		t.Fatalf("unknown run in own org should be empty, not error: %v", errsForge)
	}
}
