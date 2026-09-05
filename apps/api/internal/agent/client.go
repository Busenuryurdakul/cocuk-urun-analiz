package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	internalTokenHeader = "X-Miyuna-Internal-Token"
	defaultIPCTimeout   = 15 * time.Second
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
		return fmt.Errorf("%w: status %d", ErrOrchestratorUnavailable, resp.StatusCode)
	}
	return nil
}

func (c *OrchestratorClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
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
		return fmt.Errorf("%w: %v", ErrOrchestratorUnavailable, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d body %s", ErrOrchestratorUnavailable, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}
