package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	internalTokenHeader          = "X-Miyuna-Internal-Token"
	defaultIPCTimeout            = 60 * time.Second
	orchestratorRetryAttempts    = 3
	orchestratorRetryDelay       = 4 * time.Second
	defaultWarmupAttempts        = 45
	defaultWarmupDelay           = 6 * time.Second
	orchestratorWarmupReqTimeout = 20 * time.Second
)

// OrchestratorClient calls the Python Agent Orchestrator (internal only).
type OrchestratorClient struct {
	BaseURL        string
	Token          string
	HTTPClient     *http.Client
	WarmupAttempts int
	WarmupDelay    time.Duration
}

func NewOrchestratorClient(baseURL, token string, timeout time.Duration) *OrchestratorClient {
	if timeout <= 0 {
		timeout = defaultIPCTimeout
	}
	return &OrchestratorClient{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		WarmupAttempts: defaultWarmupAttempts,
		WarmupDelay:    defaultWarmupDelay,
	}
}

func (c *OrchestratorClient) warmupAttempts() int {
	if c == nil || c.WarmupAttempts <= 0 {
		return defaultWarmupAttempts
	}
	return c.WarmupAttempts
}

func (c *OrchestratorClient) warmupDelay() time.Duration {
	if c == nil || c.WarmupDelay <= 0 {
		return defaultWarmupDelay
	}
	return c.WarmupDelay
}

// KickWake triggers a Render cold-start without blocking the caller.
func (c *OrchestratorClient) KickWake() {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		for _, path := range []string{"/health", "/ready"} {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
			if err != nil {
				return
			}
			if c.Token != "" {
				req.Header.Set(internalTokenHeader, c.Token)
			}
			resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
			if resp != nil {
				_, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				return
			}
		}
	}()
}

type StartAnalysisRunRequest struct {
	OrganizationID string             `json:"organizationId"`
	AnalysisRunID  string             `json:"analysisRunId"`
	ProductID      string             `json:"productId"`
	TraceID        string             `json:"traceId"`
	ActorUserID    string             `json:"actorUserId"`
	Capabilities   CapabilitySnapshot `json:"capabilities"`
}

type CancelRunRequest struct {
	OrganizationID string `json:"organizationId"`
	AnalysisRunID  string `json:"analysisRunId"`
	TraceID        string `json:"traceId"`
}

func (c *OrchestratorClient) StartAnalysisRun(ctx context.Context, req StartAnalysisRunRequest) error {
	if err := c.EnsureReady(ctx); err != nil {
		return err
	}
	return c.post(ctx, "/internal/v1/runs/start", req, nil)
}

func (c *OrchestratorClient) CancelRun(ctx context.Context, req CancelRunRequest) error {
	return c.post(ctx, "/internal/v1/runs/cancel", req, nil)
}

func (c *OrchestratorClient) Ready(ctx context.Context) error {
	return c.ensureReady(ctx, 1, 0)
}

// EnsureReady polls the orchestrator until it responds or the warmup budget is exhausted.
// Render free-tier services can take 2–3 minutes to wake from hibernation (or redeploy).
func (c *OrchestratorClient) EnsureReady(ctx context.Context) error {
	return c.ensureReady(ctx, c.warmupAttempts(), c.warmupDelay())
}

func (c *OrchestratorClient) ensureReady(ctx context.Context, attempts int, delay time.Duration) error {
	if attempts <= 0 {
		attempts = 1
	}
	warmupClient := &http.Client{Timeout: orchestratorWarmupReqTimeout}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, delay); err != nil {
				return err
			}
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/ready", nil)
		if err != nil {
			return err
		}
		if c.Token != "" {
			httpReq.Header.Set(internalTokenHeader, c.Token)
		}
		resp, err := warmupClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrOrchestratorUnavailable, err)
			if attempt == 0 && attempts > 1 {
				log.Printf("agent orchestrator: warming up (attempt %d/%d): %v", attempt+1, attempts, err)
			}
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if attempt > 0 {
				log.Printf("agent orchestrator: ready after %d warmup attempt(s)", attempt+1)
			}
			return nil
		}
		lastErr = orchestratorHTTPError(resp.StatusCode, raw)
		if !isRetryableOrchestratorStatus(resp.StatusCode) {
			return lastErr
		}
		if attempts > 1 && (attempt == 0 || attempt+1 == attempts || (attempt+1)%10 == 0) {
			log.Printf("agent orchestrator: warming up (attempt %d/%d): status %d", attempt+1, attempts, resp.StatusCode)
		}
	}
	if lastErr != nil {
		log.Printf("agent orchestrator: warmup exhausted after %d attempts: %v", attempts, lastErr)
		return lastErr
	}
	return ErrOrchestratorUnavailable
}

func (c *OrchestratorClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < orchestratorRetryAttempts; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, orchestratorRetryDelay); err != nil {
				return err
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if c.Token != "" {
			httpReq.Header.Set(internalTokenHeader, c.Token)
		}

		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrOrchestratorUnavailable, err)
			if attempt+1 < orchestratorRetryAttempts {
				continue
			}
			return lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("%w: %v", ErrOrchestratorUnavailable, readErr)
			if attempt+1 < orchestratorRetryAttempts {
				continue
			}
			return lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(raw) > 0 {
				if err := json.Unmarshal(raw, out); err != nil {
					return err
				}
			}
			return nil
		}

		lastErr = orchestratorHTTPError(resp.StatusCode, raw)
		if isRetryableOrchestratorStatus(resp.StatusCode) && attempt+1 < orchestratorRetryAttempts {
			continue
		}
		return lastErr
	}
	if lastErr != nil {
		return lastErr
	}
	return ErrOrchestratorUnavailable
}

func orchestratorHTTPError(status int, raw []byte) error {
	detail := sanitizeOrchestratorBody(raw)
	if detail == "" {
		return fmt.Errorf("%w: status %d", ErrOrchestratorUnavailable, status)
	}
	return fmt.Errorf("%w: status %d (%s)", ErrOrchestratorUnavailable, status, detail)
}

func isRetryableOrchestratorStatus(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func sanitizeOrchestratorBody(raw []byte) string {
	body := strings.TrimSpace(string(raw))
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
		return "upstream gateway error page"
	}
	if strings.Contains(body, "@font-face") || strings.Contains(body, "text/html") {
		return "upstream gateway error page"
	}
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 120 {
		body = body[:120] + "…"
	}
	return body
}

// OrchestratorFailureReason returns a short, user-safe terminal error string.
func OrchestratorFailureReason(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ErrOrchestratorUnavailable.Error()
	}
	if len(msg) > 180 {
		msg = msg[:177] + "..."
	}
	return msg
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
