package llm

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

type CompletionRequest struct {
	ModelKey          string
	ProviderModelName string
	SystemInstruction string
	UserPrompt        string
	Timeout           time.Duration
	BaseURL           string
	APIKey            string
}

type CompletionResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	LatencyMS    int64
}

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error)
	HealthCheck(ctx context.Context, baseURL, apiKey, providerModelName string) error
}

type MockProvider struct {
	Responses map[string]string
	FailKeys  map[string]bool
}

func (p *MockProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	start := time.Now()
	if p.FailKeys != nil && p.FailKeys[req.ModelKey] {
		return CompletionResult{}, fmt.Errorf("mock provider failure for %s", req.ModelKey)
	}
	content := "mock response"
	if p.Responses != nil {
		if v, ok := p.Responses[req.ModelKey]; ok {
			content = v
		}
	}
	in := len(req.SystemInstruction)/4 + len(req.UserPrompt)/4
	out := len(content) / 4
	if in < 1 {
		in = 1
	}
	if out < 1 {
		out = 1
	}
	return CompletionResult{
		Content:      content,
		InputTokens:  in,
		OutputTokens: out,
		LatencyMS:    time.Since(start).Milliseconds(),
	}, nil
}

func (p *MockProvider) HealthCheck(ctx context.Context, baseURL, apiKey, providerModelName string) error {
	if strings.TrimSpace(apiKey) == "" && strings.TrimSpace(baseURL) == "" {
		return nil
	}
	return nil
}

type HTTPProvider struct {
	Client *http.Client
}

func NewHTTPProvider(timeout time.Duration) *HTTPProvider {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPProvider{Client: &http.Client{Timeout: timeout}}
}

func (p *HTTPProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	start := time.Now()
	if strings.TrimSpace(req.BaseURL) == "" {
		return CompletionResult{}, fmt.Errorf("provider base url missing")
	}
	body := map[string]any{
		"model": req.ProviderModelName,
		"messages": []map[string]string{
			{"role": "system", "content": req.SystemInstruction},
			{"role": "user", "content": req.UserPrompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(req.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}
	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResult{}, err
	}
	if resp.StatusCode >= 400 {
		return CompletionResult{}, fmt.Errorf("provider http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return CompletionResult{}, err
	}
	content := ""
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}
	in := parsed.Usage.PromptTokens
	out := parsed.Usage.CompletionTokens
	if in == 0 {
		in = len(req.SystemInstruction)/4 + len(req.UserPrompt)/4
	}
	if out == 0 {
		out = len(content) / 4
	}
	return CompletionResult{
		Content:      content,
		InputTokens:  in,
		OutputTokens: out,
		LatencyMS:    time.Since(start).Milliseconds(),
	}, nil
}

func (p *HTTPProvider) HealthCheck(ctx context.Context, baseURL, apiKey, providerModelName string) error {
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("provider not configured")
	}
	if strings.TrimSpace(apiKey) == "" && !ProviderAllowsEmptyAPIKey(baseURL) {
		return fmt.Errorf("provider not configured")
	}
	_, err := p.Complete(ctx, CompletionRequest{
		ProviderModelName: providerModelName,
		SystemInstruction: "health check",
		UserPrompt:        "ping",
		Timeout:           10 * time.Second,
		BaseURL:           baseURL,
		APIKey:            apiKey,
	})
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
