package agent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/reviewinsight"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestReviewAnalyzerAvailableAndPersistsInsights(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_review_exec_%d", time.Now().UnixNano())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	if err := client.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	def, ok := NewRegistry().Get("review_analyzer")
	if !ok || def.Availability != ToolAvailable {
		t.Fatalf("review_analyzer must be AVAILABLE: %+v", def)
	}

	runs := repository.NewAnalysisRunRepository(client.DB)
	insights := repository.NewReviewInsightRepository(client.DB)
	org := primitive.NewObjectID()
	product := primitive.NewObjectID()
	run := &domain.AnalysisRun{
		OrganizationID: org, ProductID: product, CreatedByUserID: primitive.NewObjectID(),
		ClientRequestID: fmt.Sprintf("rev-%d", time.Now().UnixNano()),
		Status:          domain.AnalysisStatusPending, TraceID: "rev", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	ratingLow := 1.0
	exec := &Executor{Registry: NewRegistry(), Reviews: &reviewinsight.Service{Insights: insights, Runs: runs}}
	out, err := exec.Execute(ctx, "review_analyzer", ToolInput{
		OrganizationID: org, RunID: run.ID, ProductID: product, Role: domain.RoleAnalyst,
		Resolved: ResolvedInput{MarketplaceReviews: []domain.MarketplaceReview{{
			ID: primitive.NewObjectID(), OrganizationID: org, ProductID: product,
			ReviewText: "choking hazard small parts broke after one day", Rating: &ratingLow,
		}}},
	})
	if err != nil {
		t.Fatalf("review_analyzer: %v", err)
	}
	if out.Status != "OK" {
		t.Fatalf("status=%s", out.Status)
	}
	if _, ok := out.Payload["positiveSignals"]; !ok {
		t.Fatal("missing positiveSignals")
	}
	if _, ok := out.Payload["safetySignals"]; !ok {
		t.Fatal("missing safetySignals")
	}
	stored, err := insights.LatestByAnalysisRun(ctx, org, run.ID)
	if err != nil {
		t.Fatalf("persist: %v", err)
	}
	if stored.ReviewCount != 1 {
		t.Fatalf("review count=%d", stored.ReviewCount)
	}
	if len(stored.SafetySignals) == 0 {
		t.Fatal("expected safety signal")
	}
}

func TestReviewAnalyzerRejectsCrossOrgReviews(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_review_xorg_%d", time.Now().UnixNano())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	runs := repository.NewAnalysisRunRepository(client.DB)
	insights := repository.NewReviewInsightRepository(client.DB)
	orgA := primitive.NewObjectID()
	orgB := primitive.NewObjectID()
	product := primitive.NewObjectID()
	run := &domain.AnalysisRun{
		OrganizationID: orgA, ProductID: product, CreatedByUserID: primitive.NewObjectID(),
		ClientRequestID: fmt.Sprintf("x-%d", time.Now().UnixNano()),
		Status:          domain.AnalysisStatusPending, TraceID: "x", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	exec := &Executor{Registry: NewRegistry(), Reviews: &reviewinsight.Service{Insights: insights, Runs: runs}}
	out, err := exec.Execute(ctx, "review_analyzer", ToolInput{
		OrganizationID: orgA, RunID: run.ID, ProductID: product, Role: domain.RoleAnalyst,
		Resolved: ResolvedInput{MarketplaceReviews: []domain.MarketplaceReview{{
			ID: primitive.NewObjectID(), OrganizationID: orgB, ProductID: product,
			ReviewText: "choking hazard",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues := out.Payload["issues"].([]string)
	if len(issues) == 0 || issues[0] != "NO_REVIEWS" {
		t.Fatalf("cross-org review must be ignored, issues=%v", issues)
	}
}
