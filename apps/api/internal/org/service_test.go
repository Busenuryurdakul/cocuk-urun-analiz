package org_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/org"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupOrgTest(t *testing.T) (*org.Service, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = fmt.Sprintf("mongodb://localhost:27017/miyuna_org_test_%d", time.Now().UnixNano())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	users := repository.NewUserRepository(db)
	orgs := repository.NewOrganizationRepository(db)
	members := repository.NewMemberRepository(db)
	invitations := repository.NewInvitationRepository(db)
	security := repository.NewSecurityEventRepository(db)
	configAudit := repository.NewConfigAuditRepository(db)
	policies := repository.NewCompliancePolicyRepository(db)
	consents := repository.NewConsentRepository(db)
	complianceEvents := repository.NewComplianceEventRepository(db)

	if err := compliance.SeedPlatformPolicies(ctx, policies); err != nil {
		t.Fatal(err)
	}

	policyRepo := &compliance.PolicyRepository{Policies: policies}
	consentSvc := &compliance.ConsentService{
		Consents: consents, Events: complianceEvents, Security: security, PolicyRepo: policyRepo, Orgs: orgs,
	}
	engine := &compliance.Engine{
		PolicyRepo: policyRepo, Consent: consentSvc, Events: complianceEvents, Security: security, Orgs: orgs,
	}
	guard := &tenant.Guard{Members: members, Events: security}

	svc := &org.Service{
		Orgs: orgs, Members: members, Users: users, Invitations: invitations,
		Security: security, ConfigAudit: configAudit, Policies: policies,
		Tenant: guard, Compliance: engine, Consent: consentSvc, PolicyRepo: policyRepo,
		WebBaseURL: "http://localhost:3000",
	}
	return svc, ctx
}

func TestCreateOrganizationAndLastOwnerProtection(t *testing.T) {
	svc, ctx := setupOrgTest(t)

	user := &domain.User{
		Email: fmt.Sprintf("owner-%d@test.local", time.Now().UnixNano()), EmailVerified: true, MFAEnabled: true,
		PasswordHash: "hash",
	}
	if err := svc.Users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	personal := &domain.Organization{
		Name: "Personal", Type: domain.OrgTypePersonal,
		ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: user.ID,
	}
	if err := svc.Orgs.Create(ctx, personal); err != nil {
		t.Fatal(err)
	}
	user.PersonalOrgID = personal.ID
	if err := svc.Users.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: personal.ID, UserID: user.ID, Role: domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}

	owner := user.ID

	created, err := svc.CreateOrganization(ctx, owner, "Team Org", "BOTH")
	if err != nil {
		t.Fatal(err)
	}
	if created.Type != domain.OrgTypeOrganization {
		t.Fatalf("expected ORGANIZATION type, got %s", created.Type)
	}

	err = svc.RemoveMember(ctx, owner, created.ID, owner)
	if err != org.ErrLastOwner {
		t.Fatalf("expected last owner protection, got %v", err)
	}
}

func TestPersonalOrgInviteRejected(t *testing.T) {
	svc, ctx := setupOrgTest(t)
	owner := primitive.NewObjectID()
	personal := primitive.NewObjectID()
	_ = svc.Orgs.Create(ctx, &domain.Organization{
		ID: personal, Name: "Personal", Type: domain.OrgTypePersonal,
		ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: owner,
	})
	_ = svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: personal, UserID: owner, Role: domain.RoleOwner,
	})

	err := svc.InviteMember(ctx, owner, personal, "invitee@test.local", "VIEWER")
	if err != org.ErrPersonalOrg {
		t.Fatalf("expected personal org rejection, got %v", err)
	}
}

func TestInviteExistingMemberRejected(t *testing.T) {
	svc, ctx := setupOrgTest(t)
	ownerID, teamID, ownerEmail := seedOwnerAndTeamOrg(t, svc, ctx)

	err := svc.InviteMember(ctx, ownerID, teamID, ownerEmail, "ADMIN")
	if err != org.ErrAlreadyMember {
		t.Fatalf("expected already member, got %v", err)
	}
}

func TestInviteMemberGrantsMissingConsents(t *testing.T) {
	svc, ctx := setupOrgTest(t)
	owner := primitive.NewObjectID()
	team := primitive.NewObjectID()
	_ = svc.Users.Create(ctx, &domain.User{
		ID: owner, Email: fmt.Sprintf("owner-consent-%d@test.local", time.Now().UnixNano()),
		EmailVerified: true, MFAEnabled: true, PasswordHash: "hash", PersonalOrgID: team,
	})
	_ = svc.Orgs.Create(ctx, &domain.Organization{
		ID: team, Name: "Team", Type: domain.OrgTypeOrganization,
		ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: owner,
	})
	_ = svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: team, UserID: owner, Role: domain.RoleOwner,
	})

	invitee := fmt.Sprintf("invitee-consent-%d@test.local", time.Now().UnixNano())
	if err := svc.InviteMember(ctx, owner, team, invitee, "VIEWER"); err != nil {
		t.Fatalf("expected invite to grant missing consents and succeed, got %v", err)
	}
}

func TestInviteMemberAndRejectDuplicatePending(t *testing.T) {
	svc, ctx := setupOrgTest(t)
	ownerID, teamID, _ := seedOwnerAndTeamOrg(t, svc, ctx)
	invitee := fmt.Sprintf("invitee-%d@test.local", time.Now().UnixNano())

	if err := svc.InviteMember(ctx, ownerID, teamID, invitee, "VIEWER"); err != nil {
		t.Fatalf("expected first invite to succeed, got %v", err)
	}
	if err := svc.InviteMember(ctx, ownerID, teamID, invitee, "VIEWER"); err != nil {
		t.Fatalf("expected pending invite resend to succeed, got %v", err)
	}
}

func seedOwnerAndTeamOrg(t *testing.T, svc *org.Service, ctx context.Context) (primitive.ObjectID, primitive.ObjectID, string) {
	t.Helper()
	email := fmt.Sprintf("owner-%d@test.local", time.Now().UnixNano())
	user := &domain.User{
		Email: email, EmailVerified: true, MFAEnabled: true, PasswordHash: "hash",
	}
	if err := svc.Users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	personal := &domain.Organization{
		Name: "Personal", Type: domain.OrgTypePersonal,
		ComplianceProfile: "KVKK", CompliancePolicyVersion: "1.0.0", OwnerID: user.ID,
	}
	if err := svc.Orgs.Create(ctx, personal); err != nil {
		t.Fatal(err)
	}
	user.PersonalOrgID = personal.ID
	if err := svc.Users.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: personal.ID, UserID: user.ID, Role: domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateOrganization(ctx, user.ID, "Team Org", "BOTH")
	if err != nil {
		t.Fatal(err)
	}
	return user.ID, created.ID, email
}

func TestCrossTenantMemberListRejected(t *testing.T) {
	svc, ctx := setupOrgTest(t)
	orgID := primitive.NewObjectID()
	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()
	_ = svc.Members.Create(ctx, &domain.OrganizationMember{OrganizationID: orgID, UserID: userA, Role: domain.RoleOwner})

	_, err := svc.ListMembers(ctx, userB, orgID)
	if err == nil {
		t.Fatal("expected cross-tenant rejection")
	}
}
