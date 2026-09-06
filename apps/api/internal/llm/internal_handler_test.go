package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInternalHandlerRequiresToken(t *testing.T) {
	h := &InternalHandler{Gateway: &Gateway{}, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	req := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestInternalHandlerRejectsInvalidOrganization(t *testing.T) {
	h := &InternalHandler{Gateway: &Gateway{}, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	body := `{"organizationId":"bad","userPrompt":"hello","correlationId":"corr"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", strings.NewReader(body))
	req.Header.Set(internalTokenHeader, "secret")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalHandlerRejectsMissingUserPrompt(t *testing.T) {
	h := &InternalHandler{Gateway: &Gateway{}, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	orgID := primitive.NewObjectID()
	body := fmt.Sprintf(`{"organizationId":"%s","correlationId":"corr"}`, orgID.Hex())
	req := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", strings.NewReader(body))
	req.Header.Set(internalTokenHeader, "secret")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "userPrompt required") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestInternalHandlerCompletePersistsAgentCall(t *testing.T) {
	gw, orgID, runID, calls, cleanup := testGateway(t)
	defer cleanup()

	h := &InternalHandler{Gateway: gw, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	payload := map[string]any{
		"organizationId": orgID.Hex(),
		"analysisRunId":  runID.Hex(),
		"userId":         primitive.NewObjectID().Hex(),
		"correlationId":  "corr-agent-1",
		"idempotencyKey": "idem-1",
		"taskType":       "analysis",
		"personaKey":     domain.PersonaCarefulAnalyst,
		"userPrompt":     "Summarize product evidence.",
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", bytes.NewReader(raw))
	req.Header.Set(internalTokenHeader, "secret")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp GatewayResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.CallID == "" {
		t.Fatal("expected callId in response")
	}
	if resp.RoutingReason == "" {
		t.Fatal("expected routingReason in response")
	}
	if resp.ProviderKey == "" {
		t.Fatal("expected providerKey in response")
	}
	if resp.CorrelationID != "corr-agent-1" {
		t.Fatalf("expected correlationId echo, got %q", resp.CorrelationID)
	}
	body := strings.ToLower(rec.Body.String())
	for _, secret := range []string{"api_key", "sk-", "env:llm_primary_api_key"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked sensitive marker %q", secret)
		}
	}

	callID, err := primitive.ObjectIDFromHex(resp.CallID)
	if err != nil {
		t.Fatal(err)
	}
	call, err := calls.FindByID(context.Background(), orgID, callID)
	if err != nil {
		t.Fatalf("expected llm_calls record: %v", err)
	}
	if call.AnalysisRunID == nil || call.AnalysisRunID.Hex() != runID.Hex() {
		t.Fatalf("expected analysisRunId linkage, got %+v", call.AnalysisRunID)
	}
	if call.CorrelationID != "corr-agent-1" {
		t.Fatalf("expected stored correlationId, got %q", call.CorrelationID)
	}
	if call.ComplianceProfile != string(domain.ProfileKVKK) {
		t.Fatalf("expected KVKK compliance profile on call, got %q", call.ComplianceProfile)
	}
	if call.ComplianceReflexVersion != compliance.ComplianceReflexVersion() {
		t.Fatalf("expected reflex version %q, got %q", compliance.ComplianceReflexVersion(), call.ComplianceReflexVersion)
	}
	if call.SafetyResult != "PASS" {
		t.Fatalf("expected safetyResult PASS, got %q", call.SafetyResult)
	}
}

func TestInternalHandlerIdempotencyReturnsSameCall(t *testing.T) {
	gw, orgID, runID, _, cleanup := testGateway(t)
	defer cleanup()

	h := &InternalHandler{Gateway: gw, Token: "secret"}
	r := chi.NewRouter()
	h.Register(r)

	payload := map[string]any{
		"organizationId": orgID.Hex(),
		"analysisRunId":  runID.Hex(),
		"correlationId":  "corr-idem",
		"idempotencyKey": "idem-dup",
		"taskType":       "review",
		"personaKey":     domain.PersonaResultAnalyst,
		"userPrompt":     "Review analysis output.",
	}
	raw, _ := json.Marshal(payload)

	first := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", bytes.NewReader(raw))
	first.Header.Set(internalTokenHeader, "secret")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, first)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first call expected 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/internal/llm/v1/complete", bytes.NewReader(raw))
	second.Header.Set(internalTokenHeader, "secret")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, second)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second call expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}

	var resp1, resp2 GatewayResponse
	_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp1.CallID == "" || resp1.CallID != resp2.CallID {
		t.Fatalf("expected duplicate idempotency to reuse callId, got %q and %q", resp1.CallID, resp2.CallID)
	}
}

func testGateway(t *testing.T) (*Gateway, primitive.ObjectID, primitive.ObjectID, *repository.LLMCallRepository, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	dbName := fmt.Sprintf("llm_handler_test_%d", time.Now().UnixNano())
	uri := fmt.Sprintf("mongodb://localhost:27017/%s", dbName)

	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	db := client.DB
	cleanup := func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
		cancel()
	}

	orgs := repository.NewOrganizationRepository(db)
	compliancePolicies := repository.NewCompliancePolicyRepository(db)
	snapshots := repository.NewConfigSnapshotRepository(db)
	providers := repository.NewLLMProviderRepository(db)
	models := repository.NewLLMModelRepository(db)
	routing := repository.NewLLMRoutingPolicyRepository(db)
	personas := repository.NewLLMPersonaRepository(db)
	calls := repository.NewLLMCallRepository(db)
	usage := repository.NewLLMUsageDailyRepository(db)
	orgSettings := repository.NewLLMOrgSettingsRepository(db)

	actorID := primitive.NewObjectID()
	if err := compliance.SeedPlatformPolicies(ctx, compliancePolicies); err != nil {
		cleanup()
		t.Fatalf("seed compliance policies: %v", err)
	}
	if err := SeedPlatformDefaults(ctx, providers, models, routing, personas, actorID); err != nil {
		cleanup()
		t.Fatalf("seed: %v", err)
	}

	orgID := primitive.NewObjectID()
	runID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	if err := orgs.Create(ctx, &domain.Organization{
		ID:                      orgID,
		Name:                    "LLM Test Org",
		Type:                    domain.OrgTypeOrganization,
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: "1.0.0",
		OwnerID:                 userID,
	}); err != nil {
		cleanup()
		t.Fatalf("create org: %v", err)
	}

	mock := &MockProvider{Responses: map[string]string{
		ModelKeyCareful: "worker-ok",
		ModelKeyResult:  "review-ok",
	}}
	policyRepo := &compliance.PolicyRepository{Policies: compliancePolicies}
	gw := NewGateway(GatewayDeps{
		Orgs:        orgs,
		OrgSettings: orgSettings,
		Providers:   providers,
		Models:      models,
		Routing:     routing,
		Personas:    personas,
		Calls:       calls,
		Usage:       usage,
		Router:      &Router{},
		Enforcer: &Enforcer{
			PolicyRepo: policyRepo,
			Snapshots:  snapshots,
		},
		Mock:       mock,
		UseMock:    true,
		MaxRetries: 0,
		Timeout:    5 * time.Second,
	})
	return gw, orgID, runID, calls, cleanup
}
