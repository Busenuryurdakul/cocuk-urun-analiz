package llm

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ollamaAvailable(baseURL string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func TestGatewayCompleteWithOllama(t *testing.T) {
	baseURL := envOrDefault("LLM_PRIMARY_BASE_URL", "http://127.0.0.1:11434/v1")
	modelName := envOrDefault("LLM_PRIMARY_MODEL_NAME", "llama3.2:latest")

	if !ollamaAvailable(baseURL) {
		t.Skip("ollama OpenAI-compatible endpoint unavailable")
	}

	_ = os.Setenv("LLM_PRIMARY_BASE_URL", baseURL)
	_ = os.Setenv("LLM_SECONDARY_BASE_URL", baseURL)
	_ = os.Setenv("LLM_PRIMARY_MODEL_NAME", modelName)
	_ = os.Setenv("LLM_SECONDARY_MODEL_NAME", modelName)
	_ = os.Setenv("LLM_PRIMARY_API_KEY", "")
	_ = os.Setenv("LLM_SECONDARY_API_KEY", "")

	gw, orgID, runID, calls, cleanup := testGatewayHTTP(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	userID := primitive.NewObjectID()
	resp, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "ollama-gateway-test",
		IdempotencyKey: "ollama-gateway-test-analysis",
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Reply with the single word OK.",
	})
	if err != nil {
		t.Fatalf("complete analysis: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected non-empty content from ollama gateway")
	}
	if resp.ProviderKey != ProviderKeyPrimary {
		t.Fatalf("expected provider %s got %s", ProviderKeyPrimary, resp.ProviderKey)
	}

	call, err := calls.FindByIdempotency(ctx, orgID, "ollama-gateway-test-analysis")
	if err != nil {
		t.Fatalf("find call: %v", err)
	}
	if call.Status != domain.LLMCallSucceeded {
		t.Fatalf("expected succeeded call got %s", call.Status)
	}
}

func testGatewayHTTP(t *testing.T) (*Gateway, primitive.ObjectID, primitive.ObjectID, *repository.LLMCallRepository, func()) {
	t.Helper()
	gw, orgID, runID, calls, cleanup := testGateway(t)

	gw.deps.UseMock = false
	gw.deps.HTTP = NewHTTPProvider(2 * time.Minute)
	gw.deps.Timeout = 2 * time.Minute
	gw.deps.MaxRetries = 0

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	modelName := envOrDefault("LLM_PRIMARY_MODEL_NAME", "llama3.2:latest")
	for _, key := range []string{ModelKeyCareful, ModelKeyResult} {
		model, err := gw.deps.Models.FindByKey(ctx, key)
		if err != nil {
			t.Fatalf("find model %s: %v", key, err)
		}
		model.ProviderModelName = modelName
		if err := gw.deps.Models.Upsert(ctx, model); err != nil {
			t.Fatalf("update model %s: %v", key, err)
		}
	}

	return gw, orgID, runID, calls, cleanup
}
