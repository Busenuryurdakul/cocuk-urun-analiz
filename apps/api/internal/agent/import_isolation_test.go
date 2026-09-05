package agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
	"github.com/go-chi/chi/v5"
)

func TestPhase4ImportQueueIsolation(t *testing.T) {
	if queue.ImportQueueKey == queue.AnalysisQueueKey {
		t.Fatal("import and analysis queues must differ")
	}
	if queue.ImportQueueKey != "marketplace:import:queue" {
		t.Fatalf("unexpected import queue key: %s", queue.ImportQueueKey)
	}
}

func TestMarketplaceConsumerUnchangedScope(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(agentTestDir(t), "..", "marketplace", "consumer.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "ProcessRun") || !strings.Contains(body, "ImportJob") {
		t.Fatal("marketplace consumer scope changed unexpectedly")
	}
	if strings.Contains(body, "analysis:run:queue") || strings.Contains(body, "AnalysisQueue") {
		t.Fatal("import consumer must not consume analysis queue")
	}
}

func TestInternalHandlerRequiresToken(t *testing.T) {
	svc := &Service{Registry: NewRegistry()}
	h := &InternalHandler{Service: svc, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	req := httptest.NewRequest(http.MethodGet, "/internal/agent/v1/capabilities", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/internal/agent/v1/capabilities", nil)
	req.Header.Set(internalTokenHeader, "secret")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func agentTestDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Dir(file)
}

func TestUnavailableToolNotPlannable(t *testing.T) {
	reg := NewRegistry()
	for _, name := range []string{"import_planner", "product_normalizer", "ecommerce_fetcher", "report_generator"} {
		if reg.IsPlannable(name) {
			t.Fatalf("unavailable tool must not be plannable: %s", name)
		}
	}
}
