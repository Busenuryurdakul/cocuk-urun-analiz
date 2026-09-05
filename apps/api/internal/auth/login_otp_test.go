package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type otpRecordingMail struct {
	messages []mail.Message
}

func (r *otpRecordingMail) Send(_ context.Context, msg mail.Message) error {
	r.messages = append(r.messages, msg)
	return nil
}

func seedVerifiedMFAUser(t *testing.T, svc *Service, email, password string) *domain.User {
	t.Helper()
	ctx := context.Background()
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	user := &domain.User{
		Email:         email,
		PasswordHash:  hash,
		EmailVerified: true,
		MFAEnabled:    true,
		MFASecret:     "JBSWY3DPEHPK3PXP",
	}
	if err := svc.Users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	org := &domain.Organization{
		Name:                    "Personal Workspace",
		Type:                    domain.OrgTypePersonal,
		ComplianceProfile:       compliance.PersonalDefaultComplianceProfile,
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
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
	return user
}

func TestLoginSendsEmailOTP(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	svc.Policy.LoginEmailOTPTTL = 10 * time.Minute
	box := &otpRecordingMail{}
	svc.Mail = box

	ctx := context.Background()
	email := "login-otp-" + primitive.NewObjectID().Hex() + "@example.com"
	user := seedVerifiedMFAUser(t, svc, email, "password12")

	result, err := svc.Login(ctx, LoginRequest{
		Email:             user.Email,
		Password:          "password12",
		DeviceFingerprint: "fp-login-otp",
		Platform:          domain.DevicePlatformWeb,
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Status != LoginStatusEmailOTPRequired {
		t.Fatalf("status: got %s want EMAIL_OTP_REQUIRED", result.Status)
	}
	if len(box.messages) == 0 {
		t.Fatal("expected login OTP email")
	}
	combined := box.messages[0].Body + box.messages[0].HTMLBody
	if !strings.Contains(combined, "doğrulama") && !strings.Contains(combined, "Giriş") {
		t.Fatalf("unexpected mail: %q", combined)
	}
}

func TestVerifyLoginEmailOTPInvalidCode(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	svc.Policy.LoginEmailOTPTTL = 10 * time.Minute
	svc.Mail = &otpRecordingMail{}

	ctx := context.Background()
	email := "bad-otp-" + primitive.NewObjectID().Hex() + "@example.com"
	user := seedVerifiedMFAUser(t, svc, email, "password12")
	login, err := svc.Login(ctx, LoginRequest{
		Email:             user.Email,
		Password:          "password12",
		DeviceFingerprint: "fp-bad-otp",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err = svc.VerifyLoginEmailOTP(ctx, login.Pending.Token, "000000", "fp-bad-otp")
	if err != ErrInvalidCode {
		t.Fatalf("got %v want INVALID_CODE", err)
	}
}

func TestRevokeDevice(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()
	userID := primitive.NewObjectID()
	device := &domain.Device{
		UserID:            userID,
		DeviceFingerprint: "revoke-fp",
		Platform:          domain.DevicePlatformWeb,
		Label:             "Test",
		Verified:          true,
	}
	if err := svc.Devices.Upsert(ctx, device); err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeDevice(ctx, userID, device.ID); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Devices.FindByID(ctx, userID, device.ID)
	if err != repository.ErrNotFound {
		t.Fatalf("revoked device should not be found: %v", err)
	}
}
