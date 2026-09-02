package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/health"
)

func TestLiveness(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	health.Liveness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadiness(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	health.Readiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
