//go:build real_llm

package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRealDualModelGatewayComplete(t *testing.T) {
	primaryModel := strings.TrimSpace(os.Getenv("LLM_PRIMARY_MODEL_NAME"))
	secondaryModel := strings.TrimSpace(os.Getenv("LLM_SECONDARY_MODEL_NAME"))
	if primaryModel == "" || secondaryModel == "" {
		t.Skip("LLM_PRIMARY_MODEL_NAME and LLM_SECONDARY_MODEL_NAME required for real_llm tests")
	}
	if primaryModel == secondaryModel {
		t.Fatal("DUAL_MODEL_REAL_E2E: worker and reviewer physical models must differ")
	}

	gw, orgID, runID, calls, cleanup := testGatewayHTTP(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	userID := primitive.NewObjectID()
	worker, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "real-worker",
		IdempotencyKey: "real-worker-" + runID.Hex(),
		TaskType:       "analysis",
		PersonaKey:     domain.PersonaCarefulAnalyst,
		UserPrompt:     "Synthetic Miyuna safety check. Reply with the single word OK.",
	})
	if err != nil {
		t.Fatalf("worker real call: %v", err)
	}
	if worker.Content == "" {
		t.Fatal("worker empty content")
	}

	reviewer, err := gw.Complete(ctx, GatewayRequest{
		OrganizationID: orgID,
		UserID:         &userID,
		AnalysisRunID:  &runID,
		CorrelationID:  "real-reviewer",
		IdempotencyKey: "real-reviewer-" + runID.Hex(),
		TaskType:       "review",
		PersonaKey:     domain.PersonaResultAnalyst,
		UserPrompt:     "Confirm prior worker output is evidence-aware. Reply OK.",
	})
	if err != nil {
		t.Fatalf("reviewer real call: %v", err)
	}
	if reviewer.Content == "" {
		t.Fatal("reviewer empty content")
	}

	call, err := calls.FindByIdempotency(ctx, orgID, "real-worker-"+runID.Hex())
	if err != nil {
		t.Fatalf("persisted worker call: %v", err)
	}
	if call.Status != domain.LLMCallSucceeded {
		t.Fatalf("worker call status=%s", call.Status)
	}
	if call.RoutingReason == "" || call.ProviderKey == "" {
		t.Fatal("worker call missing routing metadata")
	}
}
