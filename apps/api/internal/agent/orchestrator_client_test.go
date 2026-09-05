package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrchestratorClientStartAnalysisRun(t *testing.T) {
	var got StartAnalysisRunRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/runs/start" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get(internalTokenHeader) != "secret" {
			t.Fatal("missing internal token")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewOrchestratorClient(srv.URL, "secret", 0)
	if err := client.StartAnalysisRun(context.Background(), StartAnalysisRunRequest{
		OrganizationID: "org",
		AnalysisRunID:  "run",
		ProductID:      "prod",
		TraceID:        "trace-1",
		ActorUserID:    "user",
		Capabilities:   CapabilitySnapshot{PlannerVersion: PythonPlannerVersion()},
	}); err != nil {
		t.Fatal(err)
	}
	if got.TraceID != "trace-1" || got.AnalysisRunID != "run" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestOrchestratorClientUnavailable(t *testing.T) {
	client := NewOrchestratorClient("http://127.0.0.1:1", "secret", 0)
	err := client.StartAnalysisRun(context.Background(), StartAnalysisRunRequest{
		OrganizationID: "org",
		AnalysisRunID:  "run",
		TraceID:        "trace",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrOrchestratorUnavailable) {
		t.Fatalf("expected orchestrator unavailable, got %v", err)
	}
}
