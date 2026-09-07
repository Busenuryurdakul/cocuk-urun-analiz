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
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubRecallAdapter struct {
	source  string
	records []recall.Record
}

func (s stubRecallAdapter) Source() string { return s.source }
func (s stubRecallAdapter) SearchRecalls(context.Context, recall.Identity) ([]recall.Record, error) {
	return s.records, nil
}

type safetyFix struct {
	ctx    context.Context
	svc    *safety.Service
	exec   *Executor
	orgA   primitive.ObjectID
	orgB   primitive.ObjectID
	actorA primitive.ObjectID
	actorB primitive.ObjectID
	prod   *domain.Product
	run    *domain.AnalysisRun
}

func setupSafety(t *testing.T, records []recall.Record) *safetyFix {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_safety_%d", time.Now().UnixNano())
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
	mappings := repository.NewProductSourceMappingRepository(db)
	runs := repository.NewAnalysisRunRepository(db)
	evSvc := &evidence.Service{
		Evidences:   repository.NewEvidenceRepository(db),
		Validations: repository.NewEvidenceClaimValidationRepository(db),
		Runs:        runs, Products: products, Tenant: &tenant.Guard{Members: members, Events: security},
	}
	svc := &safety.Service{
		Findings: repository.NewSafetyFindingRepository(db), Matches: repository.NewRecallMatchRepository(db),
		Runs: runs, Products: products, Mappings: mappings, Evidences: evSvc,
		Tenant:   &tenant.Guard{Members: members, Events: security},
		Adapters: []safety.SourceAdapter{stubRecallAdapter{source: recall.SourceCPSC, records: records}},
	}
	f := &safetyFix{
		ctx: ctx, svc: svc, exec: &Executor{Registry: NewRegistry(), Evidence: evSvc, Safety: svc},
		orgA: primitive.NewObjectID(), orgB: primitive.NewObjectID(), actorA: primitive.NewObjectID(), actorB: primitive.NewObjectID(),
	}
	_ = members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgA, UserID: f.actorA, Role: domain.RoleAnalyst})
	_ = members.Create(ctx, &domain.OrganizationMember{OrganizationID: f.orgB, UserID: f.actorB, Role: domain.RoleOwner})
	f.prod = &domain.Product{OrganizationID: f.orgA, CreatedBy: f.actorA, Name: domain.ProductFieldMeta{Value: "Baby Rattle"}, Brand: domain.ProductFieldMeta{Value: "Acme"}}
	if err := products.Create(ctx, f.prod); err != nil {
		t.Fatal(err)
	}
	if err := mappings.Create(ctx, &domain.ProductSourceMapping{OrganizationID: f.orgA, ProductID: f.prod.ID, Source: domain.MarketplaceMiyuna, Model: "X1", Brand: "Acme", UPC: "012345678905"}); err != nil {
		t.Fatal(err)
	}
	f.run = &domain.AnalysisRun{OrganizationID: f.orgA, CreatedByUserID: f.actorA, ProductID: f.prod.ID, ClientRequestID: fmt.Sprintf("s-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending, TraceID: "s", ConfigSnapshotID: primitive.NewObjectID()}
	if err := runs.Create(ctx, f.run); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSafetyAnalyzerOfficialRecall(t *testing.T) {
	f := setupSafety(t, []recall.Record{{
		Source: recall.SourceCPSC, SourceRecordID: "123", Title: "Acme recall", Brand: "Acme", ProductName: "Baby Rattle",
		Model: "X1", Manufacturer: "Acme", UPC: "012345678905", Hazard: "Choking", Reference: "https://www.cpsc.gov/r",
	}})
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	if len(findings) != 1 || findings[0]["severity"] != "CRITICAL" || findings[0]["type"] != "RECALL" {
		t.Fatalf("findings=%v", findings)
	}
}

func TestSafetyAnalyzerGUBISNoticeToEvidence(t *testing.T) {
	f := setupSafety(t, nil)
	f.svc.Adapters = []safety.SourceAdapter{stubRecallAdapter{source: recall.SourceGUBIS, records: []recall.Record{{
		Source: recall.SourceGUBIS, SourceRecordID: "G-1", Title: "Güvensiz çıngırak", Brand: "Acme",
		ProductName: "Baby Rattle", Model: "X1", Manufacturer: "Acme", UPC: "012345678905",
		Hazard: "Boğulma", Reference: "https://gubis.ticaret.gov.tr/Bildirim/1",
	}}}}
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	if len(findings) != 1 || findings[0]["severity"] != "CRITICAL" {
		t.Fatalf("findings=%v", findings)
	}
	evs, err := f.svc.Evidences.ListByAnalysisRun(f.ctx, f.orgA, f.run.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].SourceType != "OFFICIAL_SAFETY_NOTICE" {
		t.Fatalf("gubis evidence=%+v", evs)
	}
}

func TestSafetyAnalyzerWeakRecallNotCritical(t *testing.T) {
	f := setupSafety(t, []recall.Record{{
		Source: recall.SourceCPSC, SourceRecordID: "99", Title: "hose", Brand: "Other", ProductName: "garden hose",
	}})
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	for _, item := range findings {
		if item["severity"] == "CRITICAL" {
			t.Fatalf("weak match must not be critical: %v", findings)
		}
	}
}

func TestSafetyAnalyzerNoEvidenceAndUnsupported(t *testing.T) {
	f := setupSafety(t, nil)
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSafetyIssue(out.Payload["issues"].([]string), "NO_EVIDENCE") {
		t.Fatalf("issues=%v", out.Payload["issues"])
	}
	weak, err := f.svc.Evidences.CreateEvidence(f.ctx, evidence.CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.run.ID, ProductID: f.prod.ID, Source: "note", SourceType: "MANUAL",
		Claim: "maybe unsafe", Reliability: 0.2, Freshness: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err = f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{weak.ID.Hex()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSafetyIssue(out.Payload["issues"].([]string), "UNSUPPORTED_EVIDENCE") {
		t.Fatalf("issues=%v", out.Payload["issues"])
	}
}

func TestSafetyAnalyzerCrossOrgEvidence(t *testing.T) {
	f := setupSafety(t, nil)
	otherProd := &domain.Product{OrganizationID: f.orgB, CreatedBy: f.actorB, Name: domain.ProductFieldMeta{Value: "Other"}}
	if err := f.svc.Products.Create(f.ctx, otherProd); err != nil {
		t.Fatal(err)
	}
	otherRun := &domain.AnalysisRun{OrganizationID: f.orgB, CreatedByUserID: f.actorB, ProductID: otherProd.ID, ClientRequestID: fmt.Sprintf("b-%d", time.Now().UnixNano()), Status: domain.AnalysisStatusPending, TraceID: "b", ConfigSnapshotID: primitive.NewObjectID()}
	if err := f.svc.Runs.Create(f.ctx, otherRun); err != nil {
		t.Fatal(err)
	}
	foreign, err := f.svc.Evidences.CreateEvidence(f.ctx, evidence.CreateEvidenceInput{
		OrganizationID: f.orgB, AnalysisRunID: otherRun.ID, ProductID: otherProd.ID, Source: "CPSC", SourceType: "OFFICIAL_RECALL",
		Claim: "Product matched an official recall record", Reference: "https://example.test", Reliability: 0.99, Freshness: 0.8,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{foreign.ID.Hex()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSafetyIssue(out.Payload["issues"].([]string), "WRONG_TENANT") {
		t.Fatalf("issues=%v", out.Payload["issues"])
	}
	_, err = f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgB, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{},
	})
	if !errors.Is(err, evidence.ErrForeignAnalysisRun) {
		t.Fatalf("org B must not analyze org A run: %v", err)
	}
}

func TestSafetyAnalyzerHighSeveritySupportedEvidence(t *testing.T) {
	f := setupSafety(t, nil)
	ev, err := f.svc.Evidences.CreateEvidence(f.ctx, evidence.CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.run.ID, ProductID: f.prod.ID, Source: "manual", SourceType: "MANUAL",
		Claim: "Product has a choking hazard from small parts", Reliability: 0.86, Freshness: 0.7,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{ev.ID.Hex()},
	})
	if err != nil {
		t.Fatal(err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	if len(findings) != 1 || findings[0]["type"] != "CHOKING" || findings[0]["severity"] != "HIGH" {
		t.Fatalf("findings=%v", findings)
	}
}

func TestSafetyAnalyzerOfficialPersistedEvidence(t *testing.T) {
	f := setupSafety(t, nil)
	ev, err := f.svc.Evidences.CreateEvidence(f.ctx, evidence.CreateEvidenceInput{
		OrganizationID: f.orgA, AnalysisRunID: f.run.ID, ProductID: f.prod.ID, Source: "CPSC", SourceType: "OFFICIAL_RECALL",
		Claim: "Product matched an official recall record", Reference: "https://www.cpsc.gov/r", Reliability: 0.99, Freshness: 0.8,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.exec.Execute(f.ctx, "safety_analyzer", ToolInput{
		OrganizationID: f.orgA, RunID: f.run.ID, ProductID: f.prod.ID, Role: domain.RoleAnalyst, EvidenceIDs: []string{ev.ID.Hex()},
	})
	if err != nil {
		t.Fatal(err)
	}
	findings := out.Payload["findings"].([]map[string]any)
	if len(findings) != 1 || findings[0]["severity"] != "CRITICAL" || findings[0]["type"] != "RECALL" {
		t.Fatalf("findings=%v", findings)
	}
}

func TestSafetyAnalyzerRegistryAvailable(t *testing.T) {
	def, ok := NewRegistry().Get("safety_analyzer")
	if !ok || def.Availability != ToolAvailable {
		t.Fatalf("safety_analyzer must be AVAILABLE: %+v", def)
	}
}

func containsSafetyIssue(issues []string, want string) bool {
	for _, issue := range issues {
		if issue == want {
			return true
		}
	}
	return false
}
