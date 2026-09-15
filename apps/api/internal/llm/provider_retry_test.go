package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHTTPProviderMaps429ToRateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Model busy, retry later","code":"engine_overloaded"}`))
	}))
	defer srv.Close()

	p := NewHTTPProvider(time.Second)
	_, err := p.Complete(context.Background(), CompletionRequest{
		BaseURL:           srv.URL,
		ProviderModelName: "test-model",
		SystemInstruction: "system",
		UserPrompt:        "hello",
	})
	if !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("expected ErrProviderRateLimited, got %v", err)
	}
}

func TestGatewayRetriesProviderRateLimit(t *testing.T) {
	gw, orgID, runID, _, cleanup := testGateway(t)
	defer cleanup()

	gw.deps.UseMock = true
	gw.deps.MaxRetries = 0
	gw.deps.Mock = &MockProvider{
		Responses: map[string]string{
			ModelKeyResult: "review-ok",
		},
		RateLimitRemaining: map[string]int{
			ModelKeyResult: 2,
		},
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()
	resp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "rate-limit-retry",
		IdempotencyKey: "rate-limit-retry-1",
		TaskType:       "review",
		PersonaKey:     domain.PersonaResultAnalyst,
		UserPrompt:     "Review output.",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Content != "review-ok" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
}
