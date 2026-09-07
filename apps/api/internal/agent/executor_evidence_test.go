package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type validatorFixture struct {
	exec   *Executor
	svc    *evidence.Service
	ctx    context.Context
	orgA   primitive.ObjectID
	orgB   primitive.ObjectID
	runA   *domain.AnalysisRun
	runB   *domain.AnalysisRun
	prodA  *domain.Product
	evid   *repository.EvidenceRepository
	valRep *repository.EvidenceClaimValidationRepository
}

func setupValidatorFixture(t *testing.T) *validatorFixture {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_evidence_exec_%d", time.Now().UnixNano())
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
	evid := repository.NewEvidenceRepository(db)
	vals := repository.NewEvidenceClaimValidationRepository(db)
	svc := &evidence.Service{
		Evidences: evid, Validations: vals, Runs: runs, Products: products,
		Tenant: &tenant.Guard{Members: members, Events: security},
	}
	f := &validatorFixture{
		exec:   &Executor{Registry: NewRegistry(), Evidence: svc},
		svc:    svc,
		ctx:    ctx,
		orgA:   primitive.NewObjectID(),
		orgB:   primitive.NewObjectID(),
		evid:   evid,
		valRep: vals,
	}
	actorA := primitive.NewObjectID()
	actorB := primitive.NewObjectID()
	_ = members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgA, UserID: actorA, Role: domain.RoleAnalyst})
	_ = members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgB, UserID: actorB, Role: domain.RoleOwner})

	f.prodA = &domain.Product{OrganizationID: f.orgA, CreatedBy: actorA, Name: domain.ProductFieldMeta{Value: "A"}}
	if err := products.Create(ctx, f.prodA); err != nil {
		t.Fatal(err)
	}
	prodB := &domain.Product{OrganizationID: f.orgB, CreatedBy: actorB, Name: domain.ProductFieldMeta{Value: "B"}}
	if err := products.Create(ctx, prodB); err != nil {
		t.Fatal(err)
	}
	f.runA = &domain.AnalysisRun{
		OrganizationID: f.orgA, CreatedByUserID: actorA, ProductID: f.prodA.ID,
		ClientRequestID: fmt.Sprintf("a-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending,
		TraceID: "a", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, f.runA); err != nil {
		t.Fatal(err)
	}
	f.runB = &domain.AnalysisRun{
		OrganizationID: f.orgB, CreatedByUserID: actorB, ProductID: prodB.ID,
		ClientRequestID: fmt.Sprintf("b-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending,
		TraceID: "b", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := runs.Create(ctx, f.runB); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *validatorFixture) createEvidence(t *testing.T, org, run, product primitive.ObjectID, claimID string, meta map[string]any) *domain.Evidence {
	t.Helper()
	ev, err := f.svc.CreateEvidence(f.ctx, evidence.CreateEvidenceInput{
		OrganizationID: org, AnalysisRunID: run, ProductID: product, ClaimID: claimID,
		Source: "https://example.test/src", SourceType: "URL", Claim: "Ürün boğulma riski taşıyor.",
		Snippet: "warning", Reference: "https://example.test/src", Reliability: 0.8, Freshness: 0.7, Metadata: meta,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func (f *validatorFixture) execute(t *testing.T, org, run primitive.ObjectID, claimID string, ids []string) *ToolOutput {
	t.Helper()
	out, err := f.exec.Execute(f.ctx, "evidence_validator", ToolInput{
		OrganizationID: org, RunID: run, Role: domain.RoleAnalyst,
		ClaimID: claimID, ClaimText: "Ürün boğulma riski taşıyor.", EvidenceIDs: ids,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

func assertValidator(t *testing.T, out *ToolOutput, status string, issues ...string) {
	t.Helper()
	if out.Status != "OK" {
		t.Fatalf("tool status %s", out.Status)
	}
	if out.Payload["status"] != status {
		t.Fatalf("expected %s, got %v payload=%v", status, out.Payload["status"], out.Payload)
	}
	rawIssues, _ := out.Payload["issues"].([]string)
	if len(issues) == 0 && len(rawIssues) != 0 {
		t.Fatalf("expected no issues, got %v", rawIssues)
	}
	for _, want := range issues {
		found := false
		for _, got := range rawIssues {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing issue %s in %v", want, rawIssues)
		}
	}
}

func TestEvidenceValidatorSupported(t *testing.T) {
	f := setupValidatorFixture(t)
	ev := f.createEvidence(t, f.orgA, f.runA.ID, f.prodA.ID, "claim-123", nil)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-123", []string{ev.ID.Hex()})
	assertValidator(t, out, "SUPPORTED")
	stored, err := f.valRep.FindByClaim(f.ctx, f.orgA, f.runA.ID, "claim-123")
	if err != nil {
		t.Fatal(err)
	}
	if stored.SupportStatus != domain.EvidenceSupported {
		t.Fatalf("validation not persisted: %+v", stored)
	}
}

func TestEvidenceValidatorUnsupportedNoEvidence(t *testing.T) {
	f := setupValidatorFixture(t)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-empty", []string{})
	assertValidator(t, out, "UNSUPPORTED", domain.EvidenceIssueNoEvidence)
}

func TestEvidenceValidatorInvalidEvidenceID(t *testing.T) {
	f := setupValidatorFixture(t)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-bad-id", []string{"not-an-id"})
	assertValidator(t, out, "UNSUPPORTED", domain.EvidenceIssueInvalidEvidenceID)
}

func TestEvidenceValidatorWrongTenant(t *testing.T) {
	f := setupValidatorFixture(t)
	foreign := f.createEvidence(t, f.orgB, f.runB.ID, f.runB.ProductID, "claim-foreign", nil)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-foreign", []string{foreign.ID.Hex()})
	assertValidator(t, out, "UNSUPPORTED", domain.EvidenceIssueWrongTenant)
}

func TestEvidenceValidatorWrongAnalysisRun(t *testing.T) {
	f := setupValidatorFixture(t)
	otherRun := &domain.AnalysisRun{
		OrganizationID: f.orgA, CreatedByUserID: primitive.NewObjectID(), ProductID: f.prodA.ID,
		ClientRequestID: fmt.Sprintf("other-run-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending,
		TraceID: "other", ConfigSnapshotID: primitive.NewObjectID(),
	}
	if err := f.svc.Runs.Create(f.ctx, otherRun); err != nil {
		t.Fatal(err)
	}
	ev := f.createEvidence(t, f.orgA, otherRun.ID, f.prodA.ID, "claim-run", nil)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-run", []string{ev.ID.Hex()})
	assertValidator(t, out, "UNSUPPORTED", domain.EvidenceIssueWrongAnalysisRun)
}

func TestEvidenceValidatorContradicted(t *testing.T) {
	f := setupValidatorFixture(t)
	good := f.createEvidence(t, f.orgA, f.runA.ID, f.prodA.ID, "claim-mix", nil)
	bad := f.createEvidence(t, f.orgA, f.runA.ID, f.prodA.ID, "claim-mix", map[string]any{"contradictsClaim": true})
	out := f.execute(t, f.orgA, f.runA.ID, "claim-mix", []string{good.ID.Hex(), bad.ID.Hex()})
	assertValidator(t, out, "CONTRADICTED", domain.EvidenceIssueContradictory)
}

func TestEvidenceValidatorPartiallySupported(t *testing.T) {
	f := setupValidatorFixture(t)
	good := f.createEvidence(t, f.orgA, f.runA.ID, f.prodA.ID, "claim-partial", nil)
	out := f.execute(t, f.orgA, f.runA.ID, "claim-partial", []string{good.ID.Hex(), "not-an-id"})
	assertValidator(t, out, "PARTIALLY_SUPPORTED", domain.EvidenceIssueInvalidEvidenceID)
}

func TestEvidenceValidatorInvalidReliabilityFreshness(t *testing.T) {
	f := setupValidatorFixture(t)
	now := time.Now().UTC()
	badRel := &domain.Evidence{
		OrganizationID: f.orgA, AnalysisRunID: f.runA.ID, ProductID: f.prodA.ID, ClaimID: "claim-rel",
		Source: "https://example.test", SourceType: "URL", Claim: "c", Reference: "https://example.test",
		Reliability: 1.5, Freshness: 0.5, RetrievedAt: now,
	}
	if err := f.evid.Create(f.ctx, badRel); err != nil {
		t.Fatal(err)
	}
	out := f.execute(t, f.orgA, f.runA.ID, "claim-rel", []string{badRel.ID.Hex()})
	assertValidator(t, out, "UNSUPPORTED", domain.EvidenceIssueInvalidReliability)

	badFresh := &domain.Evidence{
		OrganizationID: f.orgA, AnalysisRunID: f.runA.ID, ProductID: f.prodA.ID, ClaimID: "claim-fresh",
		Source: "https://example.test", SourceType: "URL", Claim: "c", Reference: "https://example.test",
		Reliability: 0.5, Freshness: -1, RetrievedAt: now,
	}
	if err := f.evid.Create(f.ctx, badFresh); err != nil {
		t.Fatal(err)
	}
	out2 := f.execute(t, f.orgA, f.runA.ID, "claim-fresh", []string{badFresh.ID.Hex()})
	assertValidator(t, out2, "UNSUPPORTED", domain.EvidenceIssueInvalidFreshness)
}

func TestEvidenceValidatorRejectsOrgBValidationOfOrgA(t *testing.T) {
	f := setupValidatorFixture(t)
	ev := f.createEvidence(t, f.orgA, f.runA.ID, f.prodA.ID, "claim-cross", nil)
	_, err := f.exec.Execute(f.ctx, "evidence_validator", ToolInput{
		OrganizationID: f.orgB, RunID: f.runA.ID, Role: domain.RoleAnalyst,
		ClaimID: "claim-cross", ClaimText: "Ürün boğulma riski taşıyor.", EvidenceIDs: []string{ev.ID.Hex()},
	})
	if !errors.Is(err, evidence.ErrForeignAnalysisRun) {
		t.Fatalf("org B must not validate org A run, got %v", err)
	}
}

func TestEvidenceValidatorInputSchema(t *testing.T) {
	f := setupValidatorFixture(t)
	_, err := f.exec.Execute(f.ctx, "evidence_validator", ToolInput{
		OrganizationID: f.orgA, RunID: f.runA.ID, Role: domain.RoleAnalyst,
		ClaimText: "text", EvidenceIDs: []string{},
	})
	if !errors.Is(err, ErrSchemaValidation) {
		t.Fatalf("expected schema validation, got %v", err)
	}
}

func TestEvidenceValidatorAvailableInRegistry(t *testing.T) {
	def, ok := NewRegistry().Get("evidence_validator")
	if !ok || def.Availability != ToolAvailable {
		t.Fatalf("evidence_validator must be AVAILABLE, got %+v ok=%v", def, ok)
	}
}
