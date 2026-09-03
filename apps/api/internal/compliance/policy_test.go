package compliance_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupPolicyRepo(t *testing.T) (*compliance.PolicyRepository, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/miyuna_test"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	policies := repository.NewCompliancePolicyRepository(client.DB)
	if err := compliance.SeedPlatformPolicies(ctx, policies); err != nil {
		t.Fatal(err)
	}
	return &compliance.PolicyRepository{Policies: policies}, ctx
}

func TestResolveActivePlatformFallback(t *testing.T) {
	repo, ctx := setupPolicyRepo(t)
	org := &domain.Organization{
		ID:                      primitive.NewObjectID(),
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}
	policy, err := repo.ResolveActive(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Version != compliance.PlatformDefaultPolicyVersion {
		t.Fatalf("expected %s, got %s", compliance.PlatformDefaultPolicyVersion, policy.Version)
	}
	if policy.OrganizationID != nil {
		t.Fatal("expected platform default policy")
	}
}

func TestResolveActiveOrgOverridePrecedence(t *testing.T) {
	repo, ctx := setupPolicyRepo(t)
	orgID := primitive.NewObjectID()
	override := &domain.CompliancePolicyVersion{
		OrganizationID: &orgID,
		Profile:        domain.ProfileKVKK,
		Version:        compliance.PlatformDefaultPolicyVersion,
		Status:         domain.PolicyStatusPublished,
		Rules:          compliance.DefaultPlatformPolicyRules(domain.ProfileKVKK),
		Reason:         "org override test",
		CreatedBy:      primitive.NewObjectID(),
	}
	if err := repo.Policies.Insert(ctx, override); err != nil {
		t.Fatal(err)
	}

	org := &domain.Organization{
		ID:                      orgID,
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}
	policy, err := repo.ResolveActive(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if policy.OrganizationID == nil || policy.OrganizationID.Hex() != orgID.Hex() {
		t.Fatal("expected org override policy")
	}
	if policy.Reason != "org override test" {
		t.Fatal("expected org-specific published policy")
	}
}

func TestResolveActiveRejectsLatest(t *testing.T) {
	repo, ctx := setupPolicyRepo(t)
	org := &domain.Organization{
		ID:                      primitive.NewObjectID(),
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: "latest",
	}
	_, err := repo.ResolveActive(ctx, org)
	if err != compliance.ErrInvalidPolicyVersion {
		t.Fatalf("expected invalid pin error, got %v", err)
	}
}

func TestNewPlatformVersionDoesNotChangePinnedOrgResolution(t *testing.T) {
	repo, ctx := setupPolicyRepo(t)
	orgID := primitive.NewObjectID()
	pinned := compliance.PlatformDefaultPolicyVersion
	org := &domain.Organization{
		ID:                      orgID,
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: pinned,
	}

	before, err := repo.ResolveActive(ctx, org)
	if err != nil {
		t.Fatal(err)
	}

	newVersion := &domain.CompliancePolicyVersion{
		Profile:   domain.ProfileKVKK,
		Version:   "2.0.0",
		Status:    domain.PolicyStatusPublished,
		Rules:     compliance.DefaultPlatformPolicyRules(domain.ProfileKVKK),
		Reason:    "new platform version",
		CreatedBy: primitive.NewObjectID(),
	}
	if err := repo.Policies.Insert(ctx, newVersion); err != nil {
		t.Fatal(err)
	}

	after, err := repo.ResolveActive(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != pinned || after.ID != before.ID {
		t.Fatal("pinned org resolution must not change when new platform version is published")
	}
}

func TestDraftPolicyNotResolved(t *testing.T) {
	repo, ctx := setupPolicyRepo(t)
	orgID := primitive.NewObjectID()
	draftVersion := "9.8.8-draft"
	_ = repo.Policies.Insert(ctx, &domain.CompliancePolicyVersion{
		OrganizationID: &orgID,
		Profile:        domain.ProfileKVKK,
		Version:        draftVersion,
		Status:         domain.PolicyStatusDraft,
		Rules:          compliance.DefaultPlatformPolicyRules(domain.ProfileKVKK),
		CreatedBy:      primitive.NewObjectID(),
	})
	org := &domain.Organization{
		ID:                      orgID,
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: draftVersion,
	}
	_, err := repo.ResolveActive(ctx, org)
	if err == nil {
		t.Fatal("draft policy must not resolve as active")
	}
}

func TestEngineConsentRequired(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/miyuna_test"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.DB
	orgs := repository.NewOrganizationRepository(db)
	policies := repository.NewCompliancePolicyRepository(db)
	consents := repository.NewConsentRepository(db)
	events := repository.NewComplianceEventRepository(db)
	security := repository.NewSecurityEventRepository(db)
	_ = compliance.SeedPlatformPolicies(ctx, policies)

	orgID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	_ = orgs.Create(ctx, &domain.Organization{
		ID: orgID, Name: "Test", Type: domain.OrgTypeOrganization,
		ComplianceProfile: string(domain.ProfileGDPR), CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
		OwnerID: userID,
	})

	policyRepo := &compliance.PolicyRepository{Policies: policies}
	consentSvc := &compliance.ConsentService{Consents: consents, Events: events, Security: security, PolicyRepo: policyRepo, Orgs: orgs}
	engine := &compliance.Engine{PolicyRepo: policyRepo, Consent: consentSvc, Events: events, Security: security, Orgs: orgs}

	_, err = engine.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "invite_member",
		OrganizationID: orgID,
		UserID:         userID,
		Fields:         map[string]string{"email": "a@b.com", "role": "VIEWER"},
	})
	if err != compliance.ErrConsentRequired {
		t.Fatalf("expected consent required, got %v", err)
	}
}
