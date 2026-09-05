package agent

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/alicebob/miniredis/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGrantStoreFailClosedWhenRedisUnavailable(t *testing.T) {
	store := &GrantStore{Redis: nil}
	err := store.Issue(context.Background(), &AuthorizationGrant{Nonce: "x"}, "v", "t")
	if err != ErrGrantStoreUnavailable {
		t.Fatalf("expected fail-closed, got %v", err)
	}
}

func TestGrantStoreConcurrentReplay(t *testing.T) {
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
		Nonce: "shared-token", OrganizationID: orgID, UserID: userID,
		AnalysisRunID: runID, ToolExecutionID: primitive.NewObjectID(),
		ToolName: "policy_evaluator", InputHash: "hash",
	}
	if err := store.Issue(context.Background(), grant, "v", "trace"); err != nil {
		t.Fatal(err)
	}
	binding := GrantBinding{OrganizationID: orgID, AnalysisRunID: runID, UserID: userID, ToolName: "policy_evaluator", InputHash: "hash"}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := store.Consume(context.Background(), "shared-token", binding)
			results <- err
		}()
	}
	var success, replay int
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			success++
		} else if err == ErrGrantReplay {
			replay++
		}
	}
	if success != 1 || replay != 1 {
		t.Fatalf("expected 1 success and 1 replay, got success=%d replay=%d", success, replay)
	}
}

func TestUnavailableToolSchemaRejected(t *testing.T) {
	err := ValidateToolInputHash("import_planner", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != ErrToolUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestTerminalTransitionRejected(t *testing.T) {
	if err := validateTransition(domain.AnalysisStatusCompleted, domain.AnalysisStatusRunning); err == nil {
		t.Fatal("expected invalid transition from terminal")
	}
}
