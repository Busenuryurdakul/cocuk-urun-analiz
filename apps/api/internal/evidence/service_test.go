package evidence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type evidenceFixture struct {
	svc     *Service
	ctx     context.Context
	orgA    primitive.ObjectID
	orgB    primitive.ObjectID
	actorA  primitive.ObjectID
	actorB  primitive.ObjectID
	product *domain.Product
	otherP  *domain.Product
	run     *domain.AnalysisRun
	otherR  *domain.AnalysisRun
}

func setupEvidenceTest(t *testing.T) *evidenceFixture {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_evidence_test_%d", time.Now().UnixNano())
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

	db := client.DB
	members := repository.NewMemberRepository(db)
	security := repository.NewSecurityEventRepository(db)
	products := repository.NewProductRepository(db)
	runs := repository.NewAnalysisRunRepository(db)
	svc := &Service{
		Evidences:   repository.NewEvidenceRepository(db),
		Validations: repository.NewEvidenceClaimValidationRepository(db),
		Runs:        runs,
		Products:    products,
		Tenant:      &tenant.Guard{Members: members, Events: security},
	}

	f := &evidenceFixture{
		svc:    svc,
		ctx:    ctx,
		orgA:   primitive.NewObjectID(),
		orgB:   primitive.NewObjectID(),
		actorA: primitive.NewObjectID(),
		actorB: primitive.NewObjectID(),
	}
	if err := members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgA, UserID: f.actorA, Role: domain.RoleAnalyst}); err != nil {
		t.Fatal(err)
	}
	if err := members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgB, UserID: f.actorB, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}

	f.product = &domain.Product{OrganizationID: f.orgA, CreatedBy: f.actorA, Name: domain.ProductFieldMeta{Value: "Org A Toy"}}
	if err := products.Create(ctx, f.product); err != nil {
		t.Fatal(err)
	}
	f.otherP = &domain.Product{OrganizationID: f.orgB, CreatedBy: f.actorB, Name: domain.ProductFieldMeta{Value: "Org B Toy"}}
	if err := products.Create(ctx, f.otherP); err != nil {
		t.Fatal(err)
	}

	f.run = &domain.AnalysisRun{
		OrganizationID: f.orgA, CreatedByUserID: f.actorA, ProductID: f.product.ID,
		ClientRequestID: fmt.Sprintf("run-a-%d", time.Now().UnixNano()),
		Status:          domain.AnalysisStatusPending, TraceID: "trace-a",
		ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, f.run); err != nil {
		t.Fatal(err)
	}
	f.otherR = &domain.AnalysisRun{
		OrganizationID: f.orgB, CreatedByUserID: f.actorB, ProductID: f.otherP.ID,
		ClientRequestID: fmt.Sprintf("run-b-%d", time.Now().UnixNano()),
		Status:          domain.AnalysisStatusPending, TraceID: "trace-b",
		ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, f.otherR); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *evidenceFixture) createValid(t *testing.T, claimID string) *domain.Evidence {
	t.Helper()
	ev, err := f.svc.CreateEvidence(f.ctx, CreateEvidenceInput{
		OrganizationID: f.orgA,
		AnalysisRunID:  f.run.ID,
		ProductID:      f.product.ID,
		ClaimID:        claimID,
		Source:         "https://example.test/review/1",
		SourceType:     "URL",
		Claim:          "Ürün boğulma riski taşıyor.",
		Snippet:        "small parts warning",
		Reference:      "https://example.test/review/1",
		Reliability:    0.8,
		Freshness:      0.9,
	})
	if err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}
	return ev
}

func TestCreateAndReadEvidence(t *testing.T) {
	f := setupEvidenceTest(t)
	created := f.createValid(t, "claim-123")

	got, err := f.svc.GetEvidence(f.ctx, f.orgA, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClaimID != "claim-123" || got.SourceType != "URL" {
		t.Fatalf("unexpected evidence: %+v", got)
	}

	byRun, err := f.svc.ListByAnalysisRun(f.ctx, f.orgA, f.run.ID, 50)
	if err != nil || len(byRun) != 1 {
		t.Fatalf("list by run: len=%d err=%v", len(byRun), err)
	}
	byProduct, err := f.svc.ListByProduct(f.ctx, f.orgA, f.product.ID, 50)
	if err != nil || len(byProduct) != 1 {
		t.Fatalf("list by product: len=%d err=%v", len(byProduct), err)
	}
	byClaim, err := f.svc.ListByClaim(f.ctx, f.orgA, "claim-123", 50)
	if err != nil || len(byClaim) != 1 {
		t.Fatalf("list by claim: len=%d err=%v", len(byClaim), err)
	}
}

func TestCreateRejectsInvalidReliabilityAndFreshness(t *testing.T) {
	f := setupEvidenceTest(t)
	base := CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.run.ID, ProductID: f.product.ID,
		ClaimID: "claim-rel", Source: "manual", SourceType: "MANUAL", Claim: "claim",
		Reliability: 0.5, Freshness: 0.5,
	}
	badRel := base
	badRel.Reliability = 1.2
	if _, err := f.svc.CreateEvidence(f.ctx, badRel); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid reliability, got %v", err)
	}
	badFresh := base
	badFresh.Freshness = -0.1
	if _, err := f.svc.CreateEvidence(f.ctx, badFresh); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid freshness, got %v", err)
	}
}

func TestCreateRejectsForeignRunAndProduct(t *testing.T) {
	f := setupEvidenceTest(t)
	if _, err := f.svc.CreateEvidence(f.ctx, CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.otherR.ID, ProductID: f.product.ID,
		Source: "manual", SourceType: "MANUAL", Claim: "claim", Reliability: 0.5, Freshness: 0.5,
	}); !errors.Is(err, ErrForeignAnalysisRun) {
		t.Fatalf("expected foreign analysis run, got %v", err)
	}
	if _, err := f.svc.CreateEvidence(f.ctx, CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.run.ID, ProductID: f.otherP.ID,
		Source: "manual", SourceType: "MANUAL", Claim: "claim", Reliability: 0.5, Freshness: 0.5,
	}); !errors.Is(err, ErrForeignProduct) {
		t.Fatalf("expected foreign product, got %v", err)
	}
}

func TestTenantIsolationRead(t *testing.T) {
	f := setupEvidenceTest(t)
	created := f.createValid(t, "claim-iso")

	if _, err := f.svc.GetEvidence(f.ctx, f.orgB, created.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("org B must not read org A evidence, got %v", err)
	}
	items, err := f.svc.ListByAnalysisRun(f.ctx, f.orgB, f.run.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("org B list of org A run must be empty, got %d", len(items))
	}
	_, err = f.svc.ListAnalysisEvidenceForActor(f.ctx, f.actorB, f.orgA, f.run.ID, 50)
	if !errors.Is(err, tenant.ErrCrossTenantAccess) {
		t.Fatalf("expected cross-tenant deny, got %v", err)
	}
}
