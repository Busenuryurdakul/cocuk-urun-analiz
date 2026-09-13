package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	internalTokenHeader       = "X-Miyuna-Internal-Token"
	defaultIPCTimeout         = 60 * time.Second
	orchestratorRetryAttempts = 3
	orchestratorRetryDelay    = 4 * time.Second
)

// OrchestratorClient calls the Python Agent Orchestrator (internal only).
type OrchestratorClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
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
	}
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
	return c.post(ctx, "/internal/v1/runs/start", req, nil)
}

func (c *OrchestratorClient) CancelRun(ctx context.Context, req CancelRunRequest) error {
	return c.post(ctx, "/internal/v1/runs/cancel", req, nil)
}

func (c *OrchestratorClient) Ready(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/ready", nil)
	if err != nil {
		return err
	}
	if c.Token != "" {
		httpReq.Header.Set(internalTokenHeader, c.Token)
	}
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOrchestratorUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return orchestratorHTTPError(resp.StatusCode, nil)
	}
	return nil
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
	return ErrOrchestratorUnavailable.Error()
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
