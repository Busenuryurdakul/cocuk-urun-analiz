package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestLiveGoPythonRedisMongoE2E(t *testing.T) {
	h := NewPhase5Harness(t, WithPythonOrchestrator(true))
	h.App.StartBackgroundWorkers()

	token := h.AccessToken(h.AnalystID, h.OrgID)
	clientReq := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	data, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id status traceId }
	}`, map[string]any{
		"input": map[string]any{
			"organizationId":  h.OrgID.Hex(),
			"productId":       h.ProductID.Hex(),
			"clientRequestId": clientReq,
		},
	})
	if len(errs) > 0 {
		t.Fatalf("start failed: %v", errs)
	}
	run := data["startAgentRun"].(map[string]any)
	runID := run["id"].(string)
	traceID := run["traceId"].(string)

	deadline := time.Now().Add(90 * time.Second)
	var terminal string
	var lastSeq float64
	for time.Now().Before(deadline) {
		qData, qErrs, _ := h.GraphQL(token, `query($orgId: ID!, $runId: ID!, $after: Int!) {
			agentRun(organizationId: $orgId, analysisRunId: $runId) { status currentPhase }
			agentRunEvents(organizationId: $orgId, analysisRunId: $runId, afterSequence: $after, limit: 100) { sequence phase status }
		}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID, "after": int(lastSeq)})
		if len(qErrs) > 0 {
			t.Fatalf("poll errors: %v", qErrs)
		}
		agentRun := qData["agentRun"].(map[string]any)
		terminal = agentRun["status"].(string)
		events := qData["agentRunEvents"].([]any)
		for _, raw := range events {
			ev := raw.(map[string]any)
			seq := ev["sequence"].(float64)
			if seq <= lastSeq {
				t.Fatalf("non-monotonic sequence: %v after %v", seq, lastSeq)
			}
			lastSeq = seq
		}
		if terminal == "COMPLETED" || terminal == "FAILED" || terminal == "REJECTED" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if terminal != "COMPLETED" {
		t.Fatalf("expected COMPLETED terminal state, got %s trace=%s", terminal, traceID)
	}
	if lastSeq < 3 {
		t.Fatalf("expected multiple lifecycle events, got lastSeq=%v", lastSeq)
	}
}

func TestE2ECancellationStopsRun(t *testing.T) {
	h := NewPhase5Harness(t, WithPythonOrchestrator(true))
	token := h.AccessToken(h.AnalystID, h.OrgID)
	clientReq := fmt.Sprintf("cancel-%d", time.Now().UnixNano())
	data, errs, _ := h.GraphQL(token, `mutation($input: StartAgentRunInput!) {
		startAgentRun(input: $input) { id status }
	}`, map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "productId": h.ProductID.Hex(), "clientRequestId": clientReq,
	}})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	runID := data["startAgentRun"].(map[string]any)["id"].(string)
	time.Sleep(300 * time.Millisecond)
	_, cancelErrs, _ := h.GraphQL(token, `mutation($input: CancelAgentRunInput!) {
		cancelAgentRun(input: $input) { id status }
	}`, map[string]any{"input": map[string]any{
		"organizationId": h.OrgID.Hex(), "analysisRunId": runID,
	}})
	if len(cancelErrs) > 0 {
		t.Fatalf("cancel failed: %v", cancelErrs)
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		qData, _, _ := h.GraphQL(token, `query($orgId: ID!, $runId: ID!) {
			agentRun(organizationId: $orgId, analysisRunId: $runId) { status currentPhase }
		}`, map[string]any{"orgId": h.OrgID.Hex(), "runId": runID})
		st := qData["agentRun"].(map[string]any)["status"].(string)
		if st == "REJECTED" || st == "FAILED" {
			return
		}
		time.Sleep(400 * time.Millisecond)
	}
	t.Fatal("cancelled run did not reach terminal state")
}

func TestE2EStaleLeaseRecovery(t *testing.T) {
	h := NewPhase5Harness(t)
	run := &domain.AnalysisRun{
		ID:               primitive.NewObjectID(),
		OrganizationID:   h.OrgID,
		CreatedByUserID:  h.AnalystID,
		ProductID:        h.ProductID,
		ClientRequestID:  fmt.Sprintf("stale-%d", time.Now().UnixNano()),
		Status:           domain.AnalysisStatusRunning,
		TraceID:          "stale-trace",
		ConfigSnapshotID: primitive.NewObjectID(),
	}
	past := time.Now().UTC().Add(-10 * time.Minute)
	run.StartedAt = &past
	run.LastHeartbeatAt = &past
	if err := h.App.Agent.Runs.Create(h.Ctx, run); err != nil {
		t.Fatal(err)
	}
	if h.App.RecoveryWorker == nil {
		t.Fatal("recovery worker required")
	}
	h.App.RecoveryWorker.ScanOnce(h.Ctx)
	updated, err := h.App.Agent.Runs.FindByID(h.Ctx, h.OrgID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.AnalysisStatusFailed {
		t.Fatalf("expected FAILED after stale lease, got %s", updated.Status)
	}
}
