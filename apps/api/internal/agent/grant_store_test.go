package agent

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/alicebob/miniredis/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGrantStoreAtomicConsume(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	client, err := redis.Connect(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	store := &GrantStore{Redis: client, TTL: time.Minute}
	orgID := primitive.NewObjectID()
	runID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	grant := &AuthorizationGrant{
		Nonce:           "opaque-token-value",
		OrganizationID:  orgID,
		UserID:          userID,
		AnalysisRunID:   runID,
		ToolExecutionID: primitive.NewObjectID(),
		ToolName:        "policy_evaluator",
		ToolVersion:     "frozen-v1",
		InputHash:       "abc123",
	}
	if err := store.Issue(context.Background(), grant, "reg-v1", "trace-1"); err != nil {
		t.Fatal(err)
	}
	binding := GrantBinding{
		OrganizationID: orgID,
		AnalysisRunID:  runID,
		UserID:         userID,
		ToolName:       "policy_evaluator",
		InputHash:      "abc123",
	}
	if _, err := store.Consume(context.Background(), "opaque-token-value", binding); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Consume(context.Background(), "opaque-token-value", binding); err != ErrGrantReplay {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestGrantStoreRejectsBindingMismatch(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	client, err := redis.Connect(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	store := &GrantStore{Redis: client, TTL: time.Minute}
	orgID := primitive.NewObjectID()
	runID := primitive.NewObjectID()
	grant := &AuthorizationGrant{
		Nonce:           "token-2",
		OrganizationID:  orgID,
		UserID:          primitive.NewObjectID(),
		AnalysisRunID:   runID,
		ToolExecutionID: primitive.NewObjectID(),
		ToolName:        "policy_evaluator",
		InputHash:       "hash",
	}
	if err := store.Issue(context.Background(), grant, "reg-v1", "trace"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Consume(context.Background(), "token-2", GrantBinding{
		OrganizationID: orgID,
		AnalysisRunID:  runID,
		ToolName:       "policy_evaluator",
		InputHash:      "wrong-hash",
	})
	if err != ErrAuthorizationDenied {
		t.Fatalf("expected binding mismatch, got %v", err)
	}
}

func TestRetryClassifierNonRetryableCompliance(t *testing.T) {
	decision := RetryClassifier{}.Classify(ErrComplianceRejected)
	if decision.Retryable {
		t.Fatal("compliance errors must not retry")
	}
}

func TestValidateToolOutputRequiresFields(t *testing.T) {
	err := ValidateToolOutput("policy_evaluator", "OK", map[string]any{"allowed": true})
	if err == nil {
		t.Fatal("expected schema validation failure")
	}
}
