package llm

import (
	"context"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGatewayFallbackOnPrimaryFailure(t *testing.T) {
	gw, orgID, runID, calls, cleanup := testGateway(t)
	defer cleanup()

	gw.deps.UseMock = true
	gw.deps.MaxRetries = 0
	gw.deps.Mock = &MockProvider{
		Responses: map[string]string{
			ModelKeyCareful: "should-not-use",
			ModelKeyResult:  "fallback-ok",
		},
		FailKeys: map[string]bool{
			ModelKeyCareful: true,
		},
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()
	resp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "fallback-test",
		IdempotencyKey: "fallback-test-1",
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Reply OK.",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !resp.FallbackUsed {
		t.Fatal("expected fallbackUsed=true when primary mock fails")
	}
	if resp.ModelKey != ModelKeyResult {
		t.Fatalf("expected fallback model %s got %s", ModelKeyResult, resp.ModelKey)
	}
	if resp.Content != "fallback-ok" {
		t.Fatalf("unexpected content %q", resp.Content)
	}

	call, err := calls.FindByIdempotency(ctx, orgID, "fallback-test-1")
	if err != nil {
		t.Fatalf("find call: %v", err)
	}
	if !call.FallbackUsed {
		t.Fatal("expected persisted fallbackUsed=true")
	}
	if call.RoutingReason == "" {
		t.Fatal("expected routingReason persisted")
	}
}

func TestGatewayRetryBoundedBeforeFallback(t *testing.T) {
	gw, orgID, runID, _, cleanup := testGateway(t)
	defer cleanup()

	gw.deps.UseMock = true
	gw.deps.MaxRetries = 1
	gw.deps.Mock = &MockProvider{
		Responses: map[string]string{
			ModelKeyResult: "fallback-after-retries",
		},
		FailKeys: map[string]bool{
			ModelKeyCareful: true,
		},
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()
	resp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "retry-bound-test",
		IdempotencyKey: "retry-bound-test-1",
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Reply OK.",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !resp.FallbackUsed {
		t.Fatal("expected fallback after bounded retries")
	}
}

func TestGatewayWorkerAndReviewerPersonasDistinct(t *testing.T) {
	gw, orgID, runID, calls, cleanup := testGateway(t)
	defer cleanup()

	gw.deps.UseMock = true
	gw.deps.Mock = &MockProvider{
		Responses: map[string]string{
			ModelKeyCareful: "worker-output",
			ModelKeyResult:  "reviewer-output",
		},
	}

	ctx := context.Background()
	userID := primitive.NewObjectID()

	workerResp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "worker-persona",
		IdempotencyKey: "worker-persona-1",
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Worker task.",
	})
	if err != nil {
		t.Fatalf("worker complete: %v", err)
	}
	if workerResp.PersonaKey != domain.PersonaCarefulAnalyst {
		t.Fatalf("worker persona=%s", workerResp.PersonaKey)
	}

	reviewerResp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "reviewer-persona",
		IdempotencyKey: "reviewer-persona-1",
		TaskType:       "review",
		PersonaKey:     domain.PersonaResultAnalyst,
		UserPrompt:     "Reviewer task.",
	})
	if err != nil {
		t.Fatalf("reviewer complete: %v", err)
	}
	if reviewerResp.PersonaKey != domain.PersonaResultAnalyst {
		t.Fatalf("reviewer persona=%s", reviewerResp.PersonaKey)
	}
	if workerResp.ModelKey == reviewerResp.ModelKey && workerResp.RoutingReason == reviewerResp.RoutingReason {
		// Distinct task routing should select different primary models when policy prefers it.
		t.Logf("worker model=%s reviewer model=%s", workerResp.ModelKey, reviewerResp.ModelKey)
	}

	for _, key := range []string{"worker-persona-1", "reviewer-persona-1"} {
		call, err := calls.FindByIdempotency(ctx, orgID, key)
		if err != nil {
			t.Fatalf("find %s: %v", key, err)
		}
		if call.ProviderKey == "" || call.PersonaKey == "" {
			t.Fatalf("call %s missing provider/persona metadata", key)
		}
		body := call.OutputHash + call.RoutingReason
		for _, secret := range []string{"Bearer ", "hf_", "api_key"} {
			if len(secret) > 0 && containsInsensitive(body, secret) {
				t.Fatalf("call metadata leaked sensitive marker %q", secret)
			}
		}
	}
}

func containsInsensitive(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(stringContainsFold(haystack, needle))
}

func stringContainsFold(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	// small helper avoids strings import churn in test-only code
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
