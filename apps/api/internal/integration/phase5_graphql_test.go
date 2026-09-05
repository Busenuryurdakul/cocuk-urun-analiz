package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGraphQLAnalystCanStartRun(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	clientReq := fmt.Sprintf("req-%d", time.Now().UnixNano())
	data, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id status clientRequestId }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OrgID.Hex(),
			"productId":       h.ProductID.Hex(),
			"clientRequestId": clientReq,
		},
	})
	if len(errs) > 0 {
		t.Fatalf("unexpected graphql errors: %v", errs)
	}
	run := data["startAgentRun"].(map[string]any)
	if run["status"] != "PENDING" {
		t.Fatalf("expected PENDING, got %v", run["status"])
	}
}

func TestGraphQLViewerStartRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.ViewerID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OrgID.Hex(),
			"productId":       h.ProductID.Hex(),
			"clientRequestId": fmt.Sprintf("viewer-%d", time.Now().UnixNano()),
		},
	})
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN, got %v", errs)
	}
}

func TestGraphQLViewerCanReadRuns(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.ViewerID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `query($orgId: ID!) {
		analysisRuns(organizationId: $orgId) { id status }
	}`, map[string]any{"orgId": h.OrgID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("viewer read should be allowed: %v", errs)
	}
}

func TestGraphQLCrossTenantStartRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OtherOrgID.Hex(),
			"productId":       h.OtherProdID.Hex(),
			"clientRequestId": fmt.Sprintf("cross-%d", time.Now().UnixNano()),
		},
	})
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for cross-tenant start, got %v", errs)
	}
}

func TestGraphQLCrossTenantProductRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OrgID.Hex(),
			"productId":       h.OtherProdID.Hex(),
			"clientRequestId": fmt.Sprintf("other-prod-%d", time.Now().UnixNano()),
		},
	})
	if len(errs) == 0 {
		t.Fatal("expected error for other tenant product")
	}
}

func TestGraphQLDuplicateClientRequestIdempotent(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	clientReq := fmt.Sprintf("dup-%d", time.Now().UnixNano())
	mutation := `mutation($input: StartAgentRunInput!) { startAgentRun(input: $input) { id status } }`
	vars := map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "productId": h.ProductID.Hex(), "clientRequestId": clientReq,
	}}
	data1, errs1, _ := h.GraphQL(token, mutation, vars)
	if len(errs1) > 0 {
		t.Fatal(errs1)
	}
	data2, errs2, _ := h.GraphQL(token, mutation, vars)
	if len(errs2) > 0 {
		t.Fatal(errs2)
	}
	id1 := data1["startAgentRun"].(map[string]any)["id"]
	id2 := data2["startAgentRun"].(map[string]any)["id"]
	if id1 != id2 {
		t.Fatalf("expected same run id, got %s vs %s", id1, id2)
	}
}

func TestGraphQLOrchestratorUnavailable(t *testing.T) {
	h := NewPhase5Harness(t, WithUnavailableOrchestrator())
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OrgID.Hex(),
			"productId":       h.ProductID.Hex(),
			"clientRequestId": fmt.Sprintf("unavail-%d", time.Now().UnixNano()),
		},
	})
	if len(errs) == 0 {
		t.Fatal("expected orchestrator unavailable error")
	}
}

func TestGraphQLInvalidRunIDRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": "not-a-valid-id"})
	if len(errs) == 0 {
		t.Fatal("expected invalid id error")
	}
}

func TestInternalEndpointRejectsMissingToken(t *testing.T) {
	h := NewPhase5Harness(t)
	resp, _ := h.InternalPost("/internal/agent/v1/tools/authorize", map[string]any{}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestInternalEndpointRejectsInvalidToken(t *testing.T) {
	h := NewPhase5Harness(t)
	resp, _ := h.InternalPost("/internal/agent/v1/tools/authorize", map[string]any{
		"organizationId": h.OrgID.Hex(),
		"analysisRunId":  primitive.NewObjectID().Hex(),
		"traceId":        "trace",
		"toolName":       "policy_evaluator",
		"toolVersion":    "frozen-v1",
		"inputHash":      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"stepIndex":      0,
	}, "wrong-token")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGraphQLCrossTenantRunReadRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	otherToken := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errs, _ := h.GraphQL(otherToken, `query($orgId: ID!, $runId: ID!) {
		agentRun(organizationId: $orgId, analysisRunId: $runId) { id }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN, got %v", errs)
	}
}

func TestGraphQLCrossTenantListIsolation(t *testing.T) {
	h := NewPhase5Harness(t)
	_ = seedAnalysisRun(t, h)
	otherToken := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	data, errs, _ := h.GraphQL(otherToken, `query($orgId: ID!) {
		analysisRuns(organizationId: $orgId) { id }
	}`, map[string]any{"orgId": h.OtherOrgID.Hex()})
	if len(errs) > 0 {
		t.Fatalf("list failed: %v", errs)
	}
	runs := data["analysisRuns"].([]any)
	for _, raw := range runs {
		// other org list must not include primary org runs
		_ = raw
	}
	if len(runs) != 0 {
		t.Fatalf("expected empty list for isolated org, got %d", len(runs))
	}
}

func TestGraphQLCrossTenantEventAccessRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	trace := "trace-events"
	_ = h.App.Agent.RecordOrchestratorEvent(h.Ctx, h.OrgID, runID, trace, domain.AgentPhaseRunStarted, "", domain.AgentEventRunning, nil)
	otherToken := h.AccessToken(h.OtherUserID, h.OtherOrgID)
	_, errs, _ := h.GraphQL(otherToken, `query($orgId: ID!, $runId: ID!) {
		agentRunEvents(organizationId: $orgId, analysisRunId: $runId) { sequence }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex()})
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN, got %v", errs)
	}
}

func TestGraphQLAuthorizedCancellation(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	runID := startPendingRun(t, h, token)
	_, errs, _ := h.GraphQL(token, `mutation($input: CancelAgentRunInput!) {
		cancelAgentRun(input: $input) { id status }
	}`, map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "analysisRunId": runID,
	}})
	if len(errs) > 0 {
		t.Fatalf("authorized cancel failed: %v", errs)
	}
}

func TestGraphQLUnauthorizedCancellationRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	analystToken := h.AccessToken(h.AnalystID, h.OrgID)
	runID := startPendingRun(t, h, analystToken)
	viewerToken := h.AccessToken(h.ViewerID, h.OrgID)
	_, errs, _ := h.GraphQL(viewerToken, `mutation($input: CancelAgentRunInput!) {
		cancelAgentRun(input: $input) { id }
	}`, map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "analysisRunId": runID,
	}})
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN, got %v", errs)
	}
}

func TestGraphQLProvenanceCrossTenantRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	otherImportRunID := primitive.NewObjectID()
	otherProductID := h.OtherProdID
	importRun := &domain.MarketplaceImportRun{
		ID: otherImportRunID, OrganizationID: h.OtherOrgID, RequestedBy: h.OtherUserID,
		ProductID: &otherProductID, Status: domain.ImportSucceeded, Source: domain.MarketplaceMiyuna,
	}
	if err := h.App.Marketplace.Runs.Create(h.Ctx, importRun); err != nil {
		t.Fatal(err)
	}
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{"input": map[string]any{
		"organizationId":         h.OrgID.Hex(),
		"productId":              h.ProductID.Hex(),
		"marketplaceImportRunId": otherImportRunID.Hex(),
		"clientRequestId":        fmt.Sprintf("prov-%d", time.Now().UnixNano()),
	}})
	if len(errs) == 0 {
		t.Fatal("expected provenance/import run rejection")
	}
}

func TestOversizedEventMetadataRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := seedAnalysisRun(t, h)
	large := make(map[string]string)
	for i := 0; i < 200; i++ {
		large[fmt.Sprintf("k%d", i)] = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	}
	resp, _ := h.InternalPost("/internal/agent/v1/events", map[string]any{
		"organizationId": h.OrgID.Hex(),
		"analysisRunId":  runID.Hex(),
		"traceId":        "trace",
		"phase":          domain.AgentPhaseToolExecutionStarted,
		"status":         string(domain.AgentEventRunning),
		"metadata":       large,
	}, h.Token)
	if resp.StatusCode == http.StatusNoContent {
		t.Fatal("expected oversized metadata rejection")
	}
}

func TestGrantWrongBindingRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	grant := &agent.AuthorizationGrant{
		Nonce: "binding-token", OrganizationID: h.OrgID, UserID: h.AnalystID,
		AnalysisRunID: primitive.NewObjectID(), ToolExecutionID: primitive.NewObjectID(),
		ToolName: "policy_evaluator", ToolVersion: "frozen-v1", InputHash: "abc",
	}
	if err := h.App.Agent.Grants.Issue(h.Ctx, grant, "reg", "trace"); err != nil {
		t.Fatal(err)
	}
	wrongBinding := agent.GrantBinding{
		OrganizationID: h.OrgID, AnalysisRunID: grant.AnalysisRunID,
		UserID: h.AnalystID, ToolName: "policy_evaluator", InputHash: "wrong",
	}
	if _, err := h.App.Agent.Grants.Consume(h.Ctx, "binding-token", wrongBinding); err != agent.ErrAuthorizationDenied {
		t.Fatalf("expected binding mismatch, got %v", err)
	}
}

func TestForgedGrantRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	binding := agent.GrantBinding{
		OrganizationID: h.OrgID, AnalysisRunID: primitive.NewObjectID(),
		UserID: h.AnalystID, ToolName: "policy_evaluator", InputHash: "abc",
	}
	if _, err := h.App.Agent.Grants.Consume(h.Ctx, "nonexistent-token", binding); err != agent.ErrGrantReplay {
		t.Fatalf("expected forged grant rejection, got %v", err)
	}
}

func seedAnalysisRun(t *testing.T, h *Phase5Harness) primitive.ObjectID {
	t.Helper()
	runID := primitive.NewObjectID()
	run := &domain.AnalysisRun{
		ID: runID, OrganizationID: h.OrgID, CreatedByUserID: h.AnalystID, ProductID: h.ProductID,
		ClientRequestID: fmt.Sprintf("seed-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending,
		TraceID: "seed-trace", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := h.App.Agent.Runs.Create(h.Ctx, run); err != nil {
		t.Fatal(err)
	}
	return runID
}

func startPendingRun(t *testing.T, h *Phase5Harness, token string) string {
	t.Helper()
	data, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id }
	}`, map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "productId": h.ProductID.Hex(),
		"clientRequestId": fmt.Sprintf("cancel-%d", time.Now().UnixNano()),
	}})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	return data["startAgentRun"].(map[string]any)["id"].(string)
}

func TestGrantReplayRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	if h.App.Agent.Grants == nil {
		t.Fatal("grant store required")
	}
	grant := &agent.AuthorizationGrant{
		Nonce:           "replay-token",
		OrganizationID:  h.OrgID,
		UserID:          h.AnalystID,
		AnalysisRunID:   primitive.NewObjectID(),
		ToolExecutionID: primitive.NewObjectID(),
		ToolName:        "policy_evaluator",
		ToolVersion:     "frozen-v1",
		InputHash:       "abc",
	}
	if err := h.App.Agent.Grants.Issue(h.Ctx, grant, "reg", "trace"); err != nil {
		t.Fatal(err)
	}
	binding := agent.GrantBinding{
		OrganizationID: h.OrgID,
		AnalysisRunID:  grant.AnalysisRunID,
		UserID:         h.AnalystID,
		ToolName:       "policy_evaluator",
		InputHash:      "abc",
	}
	if _, err := h.App.Agent.Grants.Consume(h.Ctx, "replay-token", binding); err != nil {
		t.Fatal(err)
	}
	if _, err := h.App.Agent.Grants.Consume(h.Ctx, "replay-token", binding); err != agent.ErrGrantReplay {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestUnavailableToolAuthorizationRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := primitive.NewObjectID()
	trace := "trace-unavailable-tool"
	run := &domain.AnalysisRun{
		ID: runID, OrganizationID: h.OrgID, CreatedByUserID: h.AnalystID, ProductID: h.ProductID,
		ClientRequestID: fmt.Sprintf("tool-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusRunning,
		TraceID: trace, ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := h.App.Agent.Runs.Create(h.Ctx, run); err != nil {
		t.Fatal(err)
	}
	resp, body := h.InternalPost("/internal/agent/v1/tools/authorize", map[string]any{
		"organizationId": h.OrgID.Hex(),
		"analysisRunId":  runID.Hex(),
		"traceId":        trace,
		"toolName":       "import_planner",
		"toolVersion":    "frozen-v1",
		"inputHash":      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"stepIndex":      0,
	}, h.Token)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d body %s", resp.StatusCode, string(body))
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	if out["allowed"] == true {
		t.Fatal("unavailable tool must not authorize")
	}
}

func TestEventPaginationAfterSequence(t *testing.T) {
	h := NewPhase5Harness(t)
	runID := primitive.NewObjectID()
	trace := "trace-pagination"
	run := &domain.AnalysisRun{
		ID: runID, OrganizationID: h.OrgID, CreatedByUserID: h.AnalystID, ProductID: h.ProductID,
		ClientRequestID: "pagination", Status: domain.AnalysisStatusRunning, TraceID: trace,
		ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := h.App.Agent.Runs.Create(h.Ctx, run); err != nil {
		t.Fatal(err)
	}
	_ = h.App.Agent.RecordOrchestratorEvent(h.Ctx, h.OrgID, runID, trace, domain.AgentPhaseRunStarted, "", domain.AgentEventRunning, nil)
	_ = h.App.Agent.RecordOrchestratorEvent(h.Ctx, h.OrgID, runID, trace, domain.AgentPhasePlanCreated, "", domain.AgentEventCompleted, nil)

	token := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(token, `query($orgId: ID!, $runId: ID!, $after: Int!) {
		agentRunEvents(organizationId: $orgId, analysisRunId: $runId, afterSequence: $after) { sequence phase }
	}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID.Hex(), "after": 1})
	if len(errs) > 0 {
		t.Fatalf("events query failed: %v", errs)
	}
	events := data["agentRunEvents"].([]any)
	if len(events) == 0 {
		t.Fatal("expected events after sequence 1")
	}
}
