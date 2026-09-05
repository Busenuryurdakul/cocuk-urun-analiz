package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/app"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const internalToken = "phase5-integration-token"

type Phase5Harness struct {
	T           *testing.T
	Ctx         context.Context
	App         *app.App
	MongoURI    string
	RedisURL    string
	BaseURL     string
	Token       string
	OrgID       primitive.ObjectID
	AnalystID   primitive.ObjectID
	ViewerID    primitive.ObjectID
	OtherOrgID  primitive.ObjectID
	OtherUserID primitive.ObjectID
	ProductID   primitive.ObjectID
	OtherProdID primitive.ObjectID
	server      *http.Server
	listener    net.Listener
	pythonCmd   *exec.Cmd
	cleanups    []func()
}

type HarnessOption func(*harnessConfig)

type harnessConfig struct {
	orchestratorURL string
	startPython     bool
	unavailable     bool
}

func WithOrchestratorURL(url string) HarnessOption {
	return func(c *harnessConfig) { c.orchestratorURL = url }
}

func WithUnavailableOrchestrator() HarnessOption {
	return func(c *harnessConfig) { c.unavailable = true }
}

func WithPythonOrchestrator(start bool) HarnessOption {
	return func(c *harnessConfig) { c.startPython = start }
}

func NewPhase5Harness(t *testing.T, opts ...HarnessOption) *Phase5Harness {
	t.Helper()
	cfg := harnessConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	h := &Phase5Harness{T: t, Ctx: ctx, Token: internalToken}
	h.cleanups = append(h.cleanups, cancel)

	dbName := fmt.Sprintf("miyuna_phase5_%d", time.Now().UnixNano())
	h.MongoURI = fmt.Sprintf("mongodb://localhost:27017/%s", dbName)
	h.RedisURL = "redis://localhost:6379/15"

	if err := pingMongo(ctx, h.MongoURI); err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	if err := flushRedisDB(ctx, h.RedisURL); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	h.cleanups = append(h.cleanups, func() { _ = flushRedisDB(context.Background(), h.RedisURL) })

	orchURL := cfg.orchestratorURL
	if cfg.unavailable {
		orchURL = "http://127.0.0.1:1"
	} else if !cfg.startPython && orchURL == "" {
		orchURL = startStubOrchestrator(t, h)
	}

	appCfg := config.Config{
		Host:                 "127.0.0.1",
		Port:                 "0",
		MongoURI:             h.MongoURI,
		RedisURL:             h.RedisURL,
		AgentOrchestratorURL: orchURL,
		AgentInternalToken:   internalToken,
		AgentIPCTimeout:      10 * time.Second,
		GoInternalAPIURL:     "http://127.0.0.1:0",
		JWTSecret:            "phase5-integration-jwt-secret-32c",
		WebBaseURL:           "http://localhost:3000",
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      time.Hour,
		LLMUseMock:           true,
	}

	application, err := app.New(ctx, appCfg)
	if err != nil {
		t.Fatalf("app init: %v", err)
	}
	h.App = application
	h.startHTTPServer(t)

	if cfg.startPython {
		pythonPort := mustFreePort(t)
		cfg.orchestratorURL = fmt.Sprintf("http://127.0.0.1:%d", pythonPort)
		h.App.Config.AgentOrchestratorURL = cfg.orchestratorURL
		if h.App.Agent.Orchestrator != nil {
			h.App.Agent.Orchestrator.BaseURL = cfg.orchestratorURL
		}
		h.startPython(t, pythonPort)
		waitHTTP200(t, cfg.orchestratorURL+"/ready", 20*time.Second)
	}

	h.seedTenants(t)
	t.Cleanup(h.Close)
	return h
}

func startStubOrchestrator(t *testing.T, h *Phase5Harness) string {
	t.Helper()
	mux := http.NewServeMux()
	accept := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Miyuna-Internal-Token") != internalToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}
	mux.HandleFunc("/internal/v1/runs/start", accept)
	mux.HandleFunc("/internal/v1/runs/cancel", accept)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	srv := httptest.NewServer(mux)
	h.cleanups = append(h.cleanups, srv.Close)
	return srv.URL
}

func (h *Phase5Harness) Close() {
	if h.pythonCmd != nil && h.pythonCmd.Process != nil {
		_ = h.pythonCmd.Process.Kill()
	}
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.server.Shutdown(ctx)
	}
	for i := len(h.cleanups) - 1; i >= 0; i-- {
		h.cleanups[i]()
	}
}

func (h *Phase5Harness) startHTTPServer(t *testing.T) {
	t.Helper()
	resolver := &graph.Resolver{
		Auth:               h.App.Auth,
		Org:                h.App.Org,
		Consent:            h.App.Consent,
		Compliance:         h.App.Compliance,
		PolicyRepo:         h.App.PolicyRepo,
		ProductService:     h.App.Products,
		UGCService:         h.App.UGC,
		MarketplaceService: h.App.Marketplace,
		DatasetService:     h.App.Dataset,
		AgentService:       h.App.Agent,
		CookieOpts:         h.App.CookieOptions(),
	}
	r := chi.NewRouter()
	if h.App.AgentInternal != nil {
		h.App.AgentInternal.Register(r)
	}
	if h.App.LLMInternal != nil {
		h.App.LLMInternal.Register(r)
	}
	r.Handle("/graphql", graph.NewHandler(resolver, h.App.Auth))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	h.listener = ln
	h.server = &http.Server{Handler: r}
	go func() { _ = h.server.Serve(ln) }()
	h.BaseURL = "http://" + ln.Addr().String()
	h.cleanups = append(h.cleanups, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.server.Shutdown(ctx)
	})
}

func (h *Phase5Harness) startPython(t *testing.T, port int) {
	t.Helper()
	agentDir := filepath.Join(repoRoot(t), "apps", "agent")
	cmd := exec.CommandContext(h.Ctx, pyExecutable(), "-3", "-m", "uvicorn", "miyuna_agent.main:app", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = agentDir
	cmd.Env = append(os.Environ(),
		"PYTHONPATH="+filepath.Join(agentDir, "src"),
		"GO_API_URL="+h.BaseURL,
		"INTERNAL_TOKEN="+internalToken,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start python orchestrator: %v", err)
	}
	h.pythonCmd = cmd
}

func (h *Phase5Harness) seedTenants(t *testing.T) {
	t.Helper()
	ctx := h.Ctx
	if err := compliance.SeedPlatformPolicies(ctx, h.App.PolicyRepo.Policies); err != nil {
		t.Fatal(err)
	}

	h.OrgID = primitive.NewObjectID()
	h.AnalystID = primitive.NewObjectID()
	h.ViewerID = primitive.NewObjectID()
	h.OtherOrgID = primitive.NewObjectID()
	h.OtherUserID = primitive.NewObjectID()

	for _, u := range []domain.User{
		{ID: h.AnalystID, Email: fmt.Sprintf("analyst-%d@test.local", time.Now().UnixNano()), EmailVerified: true, MFAEnabled: true, PersonalOrgID: h.OrgID},
		{ID: h.ViewerID, Email: fmt.Sprintf("viewer-%d@test.local", time.Now().UnixNano()), EmailVerified: true, MFAEnabled: true, PersonalOrgID: h.OrgID},
		{ID: h.OtherUserID, Email: fmt.Sprintf("other-%d@test.local", time.Now().UnixNano()), EmailVerified: true, MFAEnabled: true, PersonalOrgID: h.OtherOrgID},
	} {
		if err := h.App.Auth.Users.Create(ctx, &u); err != nil {
			t.Fatal(err)
		}
	}

	for _, spec := range []struct {
		org  domain.Organization
		user primitive.ObjectID
		role domain.OrgRole
	}{
		{domain.Organization{ID: h.OrgID, Name: "Primary Org", Type: domain.OrgTypeOrganization, ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: h.AnalystID}, h.AnalystID, domain.RoleAnalyst},
		{domain.Organization{ID: h.OtherOrgID, Name: "Other Org", Type: domain.OrgTypeOrganization, ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: h.OtherUserID}, h.OtherUserID, domain.RoleOwner},
	} {
		if err := h.App.Org.Orgs.Create(ctx, &spec.org); err != nil {
			t.Fatal(err)
		}
		if err := h.App.Org.Members.Create(ctx, &domain.OrganizationMember{OrganizationID: spec.org.ID, UserID: spec.user, Role: spec.role}); err != nil {
			t.Fatal(err)
		}
	}
	_ = h.App.Org.Members.Create(ctx, &domain.OrganizationMember{OrganizationID: h.OrgID, UserID: h.ViewerID, Role: domain.RoleViewer})

	orgCopy := h.OrgID
	_, _ = h.App.Consent.Grant(ctx, h.AnalystID, &orgCopy, domain.ConsentPurposeDataProcessing, domain.ConsentSourceWeb, "1.0.0")

	prod, _, err := h.App.Products.Create(ctx, product.CreateInput{
		OrganizationID: h.OrgID,
		ActorID:        h.AnalystID,
		Name:           "Test Product",
		Brand:          "Brand",
		Category:       "Toys",
		Description:    "Integration product",
		Source:         domain.MarketplaceMiyuna,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.ProductID = prod.ID

	otherProd, _, err := h.App.Products.Create(ctx, product.CreateInput{
		OrganizationID: h.OtherOrgID,
		ActorID:        h.OtherUserID,
		Name:           "Other Product",
		Brand:          "Brand",
		Category:       "Toys",
		Description:    "Other tenant product",
		Source:         domain.MarketplaceMiyuna,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.OtherProdID = otherProd.ID
}

func (h *Phase5Harness) AccessToken(userID, orgID primitive.ObjectID) string {
	h.T.Helper()
	ctx := h.Ctx
	device := &domain.Device{UserID: userID, DeviceFingerprint: fmt.Sprintf("integration-%s", userID.Hex()), Platform: domain.DevicePlatformWeb, Verified: true}
	if err := h.App.Auth.Devices.Upsert(ctx, device); err != nil {
		h.T.Fatal(err)
	}
	refreshRaw, refreshHash, err := auth.NewToken()
	if err != nil {
		h.T.Fatal(err)
	}
	session := &domain.Session{
		UserID:           userID,
		OrganizationID:   orgID,
		DeviceID:         device.ID,
		FamilyID:         primitive.NewObjectID(),
		RefreshTokenHash: refreshHash,
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
	}
	if err := h.App.Auth.Sessions.Create(ctx, session); err != nil {
		h.T.Fatal(err)
	}
	access, _, err := h.App.Auth.JWT.IssueAccessToken(userID, session.ID, orgID)
	if err != nil {
		h.T.Fatal(err)
	}
	_ = refreshRaw
	return access
}

type gqlResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

func (h *Phase5Harness) GraphQL(token, query string, variables map[string]any) (map[string]any, []map[string]any, int) {
	h.T.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": variables})
	req, err := http.NewRequestWithContext(h.Ctx, http.MethodPost, h.BaseURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		h.T.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: cookies.AccessCookie, Value: token})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.T.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var parsed gqlResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		h.T.Fatalf("graphql decode: %v body=%s", err, string(raw))
	}
	var data map[string]any
	if len(parsed.Data) > 0 {
		_ = json.Unmarshal(parsed.Data, &data)
	}
	errs := make([]map[string]any, 0, len(parsed.Errors))
	for _, e := range parsed.Errors {
		var item map[string]any
		_ = json.Unmarshal(e, &item)
		errs = append(errs, item)
	}
	return data, errs, resp.StatusCode
}

func (h *Phase5Harness) InternalPost(path string, payload any, token string) (*http.Response, []byte) {
	h.T.Helper()
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(h.Ctx, http.MethodPost, h.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		h.T.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Miyuna-Internal-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.T.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, raw
}

func pingMongo(ctx context.Context, uri string) error {
	c, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		return err
	}
	defer c.Disconnect(ctx)
	return c.Ping(ctx)
}

func flushRedisDB(ctx context.Context, url string) error {
	c, err := redis.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.FlushDB(ctx)
}

func mustFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitHTTP200(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", url)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func pyExecutable() string {
	if runtime.GOOS == "windows" {
		return "py"
	}
	return "python3"
}

func gqlErrorCode(err map[string]any) string {
	ext, _ := err["extensions"].(map[string]any)
	if ext == nil {
		return ""
	}
	code, _ := ext["code"].(string)
	return code
}

func errHasCode(errs []map[string]any, code string) bool {
	for _, e := range errs {
		if gqlErrorCode(e) == code {
			return true
		}
	}
	return false
}

// Ensure httptest unused import guard
var _ = httptest.NewRecorder

func stringsContains(s, sub string) bool { return strings.Contains(s, sub) }
