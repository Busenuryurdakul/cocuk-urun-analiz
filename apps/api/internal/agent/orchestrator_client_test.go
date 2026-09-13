package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	client := NewOrchestratorClient("http://127.0.0.1:1", "secret", time.Millisecond)
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

func TestOrchestratorClientSanitizesHTMLBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Bad Gateway</body></html>"))
	}))
	defer srv.Close()

	client := NewOrchestratorClient(srv.URL, "secret", time.Second)
	err := client.StartAnalysisRun(context.Background(), StartAnalysisRunRequest{
		OrganizationID: "org",
		AnalysisRunID:  "run",
		TraceID:        "trace",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "<!DOCTYPE") || strings.Contains(err.Error(), "<html") {
		t.Fatalf("expected sanitized error, got %v", err)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestOrchestratorClientRetries502(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewOrchestratorClient(srv.URL, "secret", time.Second)
	if err := client.StartAnalysisRun(context.Background(), StartAnalysisRunRequest{
		OrganizationID: "org",
		AnalysisRunID:  "run",
		TraceID:        "trace",
	}); err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected retry, attempts=%d", attempts)
	}
}
