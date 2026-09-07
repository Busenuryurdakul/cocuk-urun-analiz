package account

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubMail struct {
	mu       sync.Mutex
	lastBody string
}

func (s *stubMail) Send(_ context.Context, msg mail.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastBody = msg.Body
	return nil
}

func (s *stubMail) lastCode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, line := range strings.Split(s.lastBody, "\n") {
		if idx := strings.LastIndex(line, ":"); idx >= 0 {
			candidate := strings.TrimSpace(line[idx+1:])
			if len(candidate) == 6 {
				return candidate
			}
		}
		for _, tok := range strings.Fields(line) {
			if len(tok) == 6 {
				allDigits := true
				for _, ch := range tok {
					if ch < '0' || ch > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					return tok
				}
			}
		}
	}
	return ""
}

func setupAccountService(t *testing.T) (*Service, *stubMail, func()) {
	t.Helper()
	uri := "mongodb://localhost:27017/miyuna_test_account_lifecycle"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongo, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	_ = mongo.EnsureIndexes(ctx)

	db := mongo.DB
	policy := auth.DefaultSecurityPolicy()
	mailer := &stubMail{}
	svc := &Service{
		Users:        repository.NewUserRepository(db),
		Orgs:         repository.NewOrganizationRepository(db),
		Members:      repository.NewMemberRepository(db),
		Devices:      repository.NewDeviceRepository(db),
		Sessions:     repository.NewSessionRepository(db),
		DeletionReqs: repository.NewAccountDeletionRepository(db),
		Consents:     repository.NewConsentRepository(db),
		AnalysisRuns: repository.NewAnalysisRunRepository(db),
		ActivityLogs: repository.NewUserActivityLogRepository(db),
		Security:     repository.NewSecurityEventRepository(db),
		Mail:         mailer,
		Policy:       policy,
	}
	cleanup := func() {
		_ = mongo.Disconnect(context.Background())
	}
	return svc, mailer, cleanup
}

func createVerifiedUser(t *testing.T, svc *Service, email string) (*domain.User, *domain.Organization) {
	t.Helper()
	if !strings.Contains(email, "+") {
		email = strings.Replace(email, "@", "+"+primitive.NewObjectID().Hex()+"@", 1)
	}
	ctx := context.Background()
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	user := &domain.User{
		Email:         email,
		PasswordHash:  hash,
		EmailVerified: true,
		MFAEnabled:    false,
	}
	if err := svc.Users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	org := &domain.Organization{
		Name:                    "Personal Workspace",
		Type:                    domain.OrgTypePersonal,
		ComplianceProfile:       "KVKK",
		CompliancePolicyVersion: "1.0.0",
		OwnerID:                 user.ID,
	}
	if err := svc.Orgs.Create(ctx, org); err != nil {
		t.Fatal(err)
	}
	user.PersonalOrgID = org.ID
	if err := svc.Users.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	member := &domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           domain.RoleOwner,
	}
	if err := svc.Members.Create(ctx, member); err != nil {
		t.Fatal(err)
	}
	return user, org
}

func TestRequestAndConfirmDeletionRevokesSessionsAndDevices(t *testing.T) {
	svc, mailer, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()

	user, org := createVerifiedUser(t, svc, "delete-me@example.com")
	device := &domain.Device{UserID: user.ID, DeviceFingerprint: "fp1", Platform: domain.DevicePlatformWeb, Verified: true}
	if err := svc.Devices.Upsert(ctx, device); err != nil {
		t.Fatal(err)
	}
	session := &domain.Session{
		UserID:           user.ID,
		DeviceID:         device.ID,
		OrganizationID:   org.ID,
		FamilyID:         primitive.NewObjectID(),
		RefreshTokenHash: auth.HashToken("refresh-token"),
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
	}
	if err := svc.Sessions.Create(ctx, session); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.RequestDeletion(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	code := mailer.lastCode()
	if code == "" {
		t.Fatal("expected deletion code in mail")
	}
	if err := svc.ConfirmDeletion(ctx, user.ID, code); err != nil {
		t.Fatal(err)
	}

	updated, err := svc.Users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.IsDeleted() {
		t.Fatal("expected user deleted")
	}
	if strings.Contains(updated.Email, "delete-me@example.com") {
		t.Fatal("expected email anonymized")
	}
	if updated.PasswordHash != "" || updated.MFASecret != "" {
		t.Fatal("expected credentials cleared")
	}

	sessions, err := svc.Sessions.ListActiveByUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected sessions revoked, got %d", len(sessions))
	}
	devices, err := svc.Devices.ListByUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 0 {
		t.Fatalf("expected devices revoked, got %d", len(devices))
	}
	if _, err := svc.Members.FindByUserAndOrg(ctx, user.ID, org.ID); err == nil {
		t.Fatal("expected membership removed")
	}
	if _, err := svc.Orgs.FindByID(ctx, org.ID); err == nil {
		t.Fatal("expected personal org deleted")
	}
}

func TestConfirmDeletionBlocksOwnerWithOtherMembers(t *testing.T) {
	svc, mailer, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()

	owner, _ := createVerifiedUser(t, svc, "owner@example.com")
	teamOrg := &domain.Organization{
		Name:                    "Team Org",
		Type:                    domain.OrgTypeOrganization,
		ComplianceProfile:       "KVKK",
		CompliancePolicyVersion: "1.0.0",
		OwnerID:                 owner.ID,
	}
	if err := svc.Orgs.Create(ctx, teamOrg); err != nil {
		t.Fatal(err)
	}
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: teamOrg.ID,
		UserID:         owner.ID,
		Role:           domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}
	memberUser, _ := createVerifiedUser(t, svc, "member@example.com")
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: teamOrg.ID,
		UserID:         memberUser.ID,
		Role:           domain.RoleAnalyst,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.RequestDeletion(ctx, owner.ID); err != nil {
		t.Fatal(err)
	}
	err := svc.ConfirmDeletion(ctx, owner.ID, mailer.lastCode())
	if err != ErrOwnershipTransferRequired {
		t.Fatalf("expected ownership block, got %v", err)
	}
	stillOwner, err := svc.Users.FindByID(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillOwner.IsDeleted() {
		t.Fatal("owner should not be deleted when blocked")
	}
}

func TestConfirmDeletionAllowsNonOwnerMember(t *testing.T) {
	svc, mailer, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()

	owner, _ := createVerifiedUser(t, svc, "team-owner@example.com")
	teamOrg := &domain.Organization{
		Name:                    "Shared Team",
		Type:                    domain.OrgTypeOrganization,
		ComplianceProfile:       "KVKK",
		CompliancePolicyVersion: "1.0.0",
		OwnerID:                 owner.ID,
	}
	if err := svc.Orgs.Create(ctx, teamOrg); err != nil {
		t.Fatal(err)
	}
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: teamOrg.ID,
		UserID:         owner.ID,
		Role:           domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}

	member, _ := createVerifiedUser(t, svc, "team-member@example.com")
	if err := svc.Members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: teamOrg.ID,
		UserID:         member.ID,
		Role:           domain.RoleViewer,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.RequestDeletion(ctx, member.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmDeletion(ctx, member.ID, mailer.lastCode()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Members.FindByUserAndOrg(ctx, member.ID, teamOrg.ID); err == nil {
		t.Fatal("expected membership removed")
	}
	if _, err := svc.Orgs.FindByID(ctx, teamOrg.ID); err != nil {
		t.Fatal("team org should remain when non-owner leaves")
	}
}

func TestDeletionCodeSingleUseAndInvalidCode(t *testing.T) {
	svc, mailer, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()
	user, _ := createVerifiedUser(t, svc, "token@example.com")

	if _, err := svc.RequestDeletion(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	code := mailer.lastCode()
	if err := svc.ConfirmDeletion(ctx, user.ID, "000000"); err != ErrInvalidDeletionCode {
		t.Fatalf("expected invalid code error, got %v", err)
	}
	if err := svc.ConfirmDeletion(ctx, user.ID, code); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmDeletion(ctx, user.ID, code); err != nil {
		t.Fatalf("second confirm after success should be idempotent, got %v", err)
	}
}

func TestExportExcludesSecretsAndOtherUsers(t *testing.T) {
	svc, _, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()

	userA, _ := createVerifiedUser(t, svc, "export-a@example.com")
	userB, _ := createVerifiedUser(t, svc, "export-b@example.com")
	_ = svc.Users.EnableMFA(ctx, userA.ID, "totp-secret-value")

	doc, err := svc.ExportData(ctx, userA.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(raw)
	if strings.Contains(payload, "totp-secret-value") {
		t.Fatal("export must not include TOTP secret")
	}
	if strings.Contains(payload, userA.PasswordHash) {
		t.Fatal("export must not include password hash")
	}
	if strings.Contains(payload, userB.Email) {
		t.Fatal("export must not include other user email")
	}
	if doc.User.Email != userA.Email {
		t.Fatal("expected own profile email")
	}
	if doc.SchemaVersion != ExportSchemaVersion {
		t.Fatalf("expected schema version %s", ExportSchemaVersion)
	}
}

func TestDeletionSecurityEventHasNoSecrets(t *testing.T) {
	svc, mailer, cleanup := setupAccountService(t)
	defer cleanup()
	ctx := context.Background()
	user, _ := createVerifiedUser(t, svc, "audit@example.com")

	if _, err := svc.RequestDeletion(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	code := mailer.lastCode()
	if err := svc.ConfirmDeletion(ctx, user.ID, code); err != nil {
		t.Fatal(err)
	}

	events, err := svc.Security.ListByUser(ctx, user.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range events {
		if ev.EventType == domain.EventAccountDeleted {
			found = true
			for _, v := range ev.Details {
				if strings.Contains(v, code) || strings.Contains(strings.ToLower(v), "password") {
					t.Fatal("security event leaked sensitive data")
				}
			}
		}
	}
	if !found {
		t.Fatal("expected ACCOUNT_DELETED security event")
	}
}
