package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrMFARequired        = errors.New("mfa setup required")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidCode        = errors.New("invalid code")
	ErrChallengeLocked    = errors.New("challenge locked")
)

const (
	StageMFA    = "MFA"
	StageDevice = "DEVICE"
)

type RegisterResult struct {
	Message string
}

type PendingInfo struct {
	Token             string
	UserID            primitive.ObjectID
	DeviceFingerprint string
	Stage             string
	ExpiresAt         time.Time
}

type MFASetupInfo struct {
	Token      string
	Secret     string
	OTPAuthURL string
	ExpiresAt  time.Time
}

type LoginResult struct {
	Status   LoginStatus
	User     *domain.User
	Pending  *PendingInfo
	MFASetup *MFASetupInfo
	Tokens   *AuthTokens
}

type LoginStatus string

const (
	LoginStatusMFASetupRequired           LoginStatus = "MFA_SETUP_REQUIRED"
	LoginStatusMFARequired                LoginStatus = "MFA_REQUIRED"
	LoginStatusDeviceVerificationRequired LoginStatus = "DEVICE_VERIFICATION_REQUIRED"
	LoginStatusAuthenticated              LoginStatus = "AUTHENTICATED"
)

type Service struct {
	Users        *repository.UserRepository
	Orgs         *repository.OrganizationRepository
	Members      *repository.MemberRepository
	Devices      *repository.DeviceRepository
	Sessions     *repository.SessionRepository
	Rotated      *repository.RotatedRefreshRepository
	Pending      *repository.PendingAuthRepository
	EmailVerify  *repository.EmailVerificationRepository
	DeviceVerify *repository.DeviceVerificationRepository
	MFASetup     *repository.MFASetupRepository
	Security     *repository.SecurityEventRepository
	Mail         mail.Service
	Tenant       *tenant.Guard
	BruteForce   *BruteForceGuard
	JWT          *JWTManager
	Policy       SecurityPolicy
	WebBaseURL   string
	MFAIssuer    string
}

const registerAckMessage = "Kayıt alındı. E-posta adresinize doğrulama bağlantısı gönderildi."

func (s *Service) Register(ctx context.Context, email, password string) (*RegisterResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 {
		return nil, ErrInvalidCredentials
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:         email,
		PasswordHash:  hash,
		EmailVerified: false,
		MFAEnabled:    false,
	}
	if err := s.Users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			// Non-enumerating response: same message, no error.
			// Unverified accounts get a fresh link so a lost mail does not block login.
			s.resendIfUnverified(ctx, email)
			return &RegisterResult{Message: registerAckMessage}, nil
		}
		return nil, err
	}

	org := &domain.Organization{
		Name:                    "Personal Workspace",
		Type:                    domain.OrgTypePersonal,
		ComplianceProfile:       compliance.PersonalDefaultComplianceProfile,
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
		OwnerID:                 user.ID,
	}
	if err := s.Orgs.Create(ctx, org); err != nil {
		return nil, err
	}

	user.PersonalOrgID = org.ID
	if err := s.Users.Update(ctx, user); err != nil {
		return nil, err
	}

	member := &domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           domain.RoleOwner,
	}
	if err := s.Members.Create(ctx, member); err != nil {
		return nil, err
	}

	if err := s.sendEmailVerification(ctx, user.ID, email); err != nil {
		return nil, err
	}

	return &RegisterResult{Message: registerAckMessage}, nil
}

func (s *Service) ResendEmailVerification(ctx context.Context, email, password string) (*RegisterResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	ack := &RegisterResult{Message: registerAckMessage}
	if email == "" || len(password) < 8 {
		return ack, nil
	}
	if s.emailResendBlocked(ctx, email) {
		return ack, nil
	}

	user, err := s.Users.FindByEmail(ctx, email)
	if err != nil {
		return ack, nil
	}
	if !CheckPassword(user.PasswordHash, password) {
		s.recordEmailResend(ctx, email, &user.ID)
		return ack, nil
	}
	if user.EmailVerified {
		return ack, nil
	}
	if s.recordEmailResend(ctx, email, &user.ID) {
		return ack, nil
	}
	if err := s.sendEmailVerification(ctx, user.ID, email); err != nil {
		return nil, err
	}
	return ack, nil
}

func (s *Service) resendIfUnverified(ctx context.Context, email string) {
	user, err := s.Users.FindByEmail(ctx, email)
	if err != nil || user.EmailVerified {
		return
	}
	if s.emailResendBlocked(ctx, email) || s.recordEmailResend(ctx, email, &user.ID) {
		return
	}
	_ = s.sendEmailVerification(ctx, user.ID, email)
}

func (s *Service) emailResendBlocked(ctx context.Context, email string) bool {
	if s.BruteForce == nil {
		return false
	}
	blocked, _ := s.BruteForce.IsOTPBlocked(ctx, "email_resend", email)
	return blocked
}

func (s *Service) recordEmailResend(ctx context.Context, email string, userID *primitive.ObjectID) bool {
	if s.BruteForce == nil {
		return false
	}
	locked, _ := s.BruteForce.RecordOTPFailure(ctx, "email_resend", email, userID)
	return locked
}

func (s *Service) sendEmailVerification(ctx context.Context, userID primitive.ObjectID, email string) error {
	token, tokenHash, err := NewToken()
	if err != nil {
		return err
	}
	ev := &domain.EmailVerification{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(s.Policy.EmailVerifyTTL),
	}
	if err := s.EmailVerify.Create(ctx, ev); err != nil {
		return err
	}
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", strings.TrimRight(s.WebBaseURL, "/"), token)
	return s.Mail.Send(ctx, mail.Message{
		To:      email,
		Subject: "Miyuna — e-posta doğrulama",
		Body:    fmt.Sprintf("E-posta adresinizi doğrulamak için bağlantıyı açın:\n%s", verifyURL),
	})
}

func (s *Service) VerifyEmail(ctx context.Context, token string) (*MFASetupInfo, error) {
	hash := HashToken(token)
	if blocked, _ := s.BruteForce.IsOTPBlocked(ctx, "email_verify", hash); blocked {
		return nil, ErrChallengeLocked
	}
	ev, err := s.EmailVerify.FindByTokenHash(ctx, hash)
	if err != nil {
		if locked, _ := s.BruteForce.RecordOTPFailure(ctx, "email_verify", hash, nil); locked {
			_ = s.EmailVerify.LockByTokenHash(ctx, hash)
		}
		return nil, ErrInvalidToken
	}
	if _, err := s.EmailVerify.ConsumeByTokenHash(ctx, hash); err != nil {
		return nil, ErrInvalidToken
	}
	_ = s.BruteForce.ResetOTPFailures(ctx, "email_verify", hash)
	if err := s.Users.SetEmailVerified(ctx, ev.UserID); err != nil {
		return nil, err
	}
	return s.beginMFASetup(ctx, ev.UserID)
}

func (s *Service) beginMFASetup(ctx context.Context, userID primitive.ObjectID) (*MFASetupInfo, error) {
	user, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	secret, url, err := GenerateTOTPSecret(s.MFAIssuer, user.Email)
	if err != nil {
		return nil, err
	}
	token, tokenHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	challenge := &domain.MFASetupChallenge{
		UserID:    userID,
		TokenHash: tokenHash,
		Secret:    secret,
		ExpiresAt: time.Now().UTC().Add(s.Policy.MFASetupTTL),
	}
	if err := s.MFASetup.Create(ctx, challenge); err != nil {
		return nil, err
	}
	return &MFASetupInfo{
		Token:      token,
		Secret:     secret,
		OTPAuthURL: url,
		ExpiresAt:  challenge.ExpiresAt,
	}, nil
}

func (s *Service) ConfirmMFA(ctx context.Context, setupToken, code string) error {
	hash := HashToken(setupToken)
	challenge, err := s.MFASetup.FindByTokenHash(ctx, hash)
	if err != nil {
		return ErrInvalidToken
	}
	if !ValidateTOTP(challenge.Secret, code) {
		uid := challenge.UserID
		if locked, _ := s.MFASetup.IncrementFailedAttempts(ctx, challenge.ID, s.Policy.MaxOTPAttempts); locked {
			_ = s.Security.Record(ctx, domain.SecurityEvent{
				UserID:    &uid,
				EventType: domain.EventOTPAttemptLimit,
				Severity:  domain.SeverityWarning,
				Details:   map[string]string{"stage": "mfa_setup"},
			})
			return ErrChallengeLocked
		}
		_, _ = s.BruteForce.RecordOTPFailure(ctx, "mfa_setup", hash, &uid)
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventMFAFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"stage": "mfa_setup"},
		})
		return ErrInvalidCode
	}
	consumed, err := s.MFASetup.ConsumeByTokenHash(ctx, hash)
	if err != nil {
		return ErrInvalidToken
	}
	_ = s.BruteForce.ResetOTPFailures(ctx, "mfa_setup", hash)
	return s.Users.EnableMFA(ctx, consumed.UserID, consumed.Secret)
}

func (s *Service) Login(ctx context.Context, email, password, deviceFingerprint string) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if locked, _ := s.BruteForce.IsLoginLocked(ctx, email); locked {
		return nil, ErrAccountLocked
	}

	user, err := s.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_, _ = s.BruteForce.RecordLoginFailure(ctx, email, nil)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !CheckPassword(user.PasswordHash, password) {
		uid := user.ID
		_, _ = s.BruteForce.RecordLoginFailure(ctx, email, &uid)
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventAuthFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"reason": "bad_password"},
		})
		return nil, ErrInvalidCredentials
	}

	_ = s.BruteForce.ResetLoginFailures(ctx, email)

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}
	if !user.MFAEnabled {
		setup, err := s.beginMFASetup(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Status: LoginStatusMFASetupRequired, User: user, MFASetup: setup}, nil
	}

	device := &domain.Device{
		UserID:            user.ID,
		DeviceFingerprint: deviceFingerprint,
		Platform:          domain.DevicePlatformWeb,
		Verified:          false,
	}
	if err := s.Devices.Upsert(ctx, device); err != nil {
		return nil, err
	}

	pendingToken, pendingHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	pending := &domain.PendingAuth{
		TokenHash:         pendingHash,
		UserID:            user.ID,
		DeviceFingerprint: deviceFingerprint,
		Stage:             StageMFA,
		ExpiresAt:         time.Now().UTC().Add(s.Policy.PendingAuthTTL),
	}
	if err := s.Pending.Create(ctx, pending); err != nil {
		return nil, err
	}

	return &LoginResult{
		Status: LoginStatusMFARequired,
		User:   user,
		Pending: &PendingInfo{
			Token:             pendingToken,
			UserID:            user.ID,
			DeviceFingerprint: deviceFingerprint,
			Stage:             StageMFA,
			ExpiresAt:         pending.ExpiresAt,
		},
	}, nil
}

func (s *Service) VerifyLoginMFA(ctx context.Context, pendingToken, code, deviceFingerprint string) (*LoginResult, error) {
	hash := HashToken(pendingToken)
	pending, err := s.Pending.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if pending.DeviceFingerprint != deviceFingerprint || pending.Stage != StageMFA {
		return nil, ErrInvalidToken
	}

	user, err := s.Users.FindByID(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}

	if !ValidateTOTP(user.MFASecret, code) {
		uid := user.ID
		if locked, _ := s.Pending.IncrementFailedAttempts(ctx, pending.ID, s.Policy.MaxOTPAttempts); locked {
			_ = s.Security.Record(ctx, domain.SecurityEvent{
				UserID:    &uid,
				EventType: domain.EventOTPAttemptLimit,
				Severity:  domain.SeverityWarning,
				Details:   map[string]string{"stage": "login_mfa"},
			})
			return nil, ErrChallengeLocked
		}
		_, _ = s.BruteForce.RecordOTPFailure(ctx, "login_mfa", hash, &uid)
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventMFAFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"stage": "login"},
		})
		return nil, ErrInvalidCode
	}
	_ = s.Pending.DeleteByTokenHash(ctx, pending.TokenHash)
	_ = s.BruteForce.ResetOTPFailures(ctx, "login_mfa", hash)

	device, err := s.Devices.FindByUserAndFingerprint(ctx, user.ID, deviceFingerprint)
	if err != nil {
		return nil, err
	}
	if device.Verified {
		tokens, err := s.issueAuthTokens(ctx, user, device)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Status: LoginStatusAuthenticated, User: user, Tokens: tokens}, nil
	}

	codePlain, codeHash, err := NewNumericCode()
	if err != nil {
		return nil, err
	}
	dv := &domain.DeviceVerification{
		UserID:    user.ID,
		DeviceID:  device.ID,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().UTC().Add(s.Policy.DeviceVerifyTTL),
	}
	if err := s.DeviceVerify.Create(ctx, dv); err != nil {
		return nil, err
	}
	_ = s.Mail.Send(ctx, mail.Message{
		To:      user.Email,
		Subject: "Miyuna — cihaz doğrulama",
		Body:    fmt.Sprintf("Yeni cihaz doğrulama kodunuz: %s", codePlain),
	})

	newPendingToken, newPendingHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	next := &domain.PendingAuth{
		TokenHash:         newPendingHash,
		UserID:            user.ID,
		DeviceFingerprint: deviceFingerprint,
		Stage:             StageDevice,
		ExpiresAt:         time.Now().UTC().Add(s.Policy.PendingAuthTTL),
	}
	if err := s.Pending.Create(ctx, next); err != nil {
		return nil, err
	}

	return &LoginResult{
		Status: LoginStatusDeviceVerificationRequired,
		User:   user,
		Pending: &PendingInfo{
			Token:             newPendingToken,
			UserID:            user.ID,
			DeviceFingerprint: deviceFingerprint,
			Stage:             StageDevice,
			ExpiresAt:         next.ExpiresAt,
		},
	}, nil
}

func (s *Service) VerifyDevice(ctx context.Context, pendingToken, code, deviceFingerprint string) (*LoginResult, error) {
	hash := HashToken(pendingToken)
	pending, err := s.Pending.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if pending.DeviceFingerprint != deviceFingerprint || pending.Stage != StageDevice {
		return nil, ErrInvalidToken
	}

	device, err := s.Devices.FindByUserAndFingerprint(ctx, pending.UserID, deviceFingerprint)
	if err != nil {
		return nil, ErrInvalidToken
	}

	dv, err := s.DeviceVerify.FindActive(ctx, pending.UserID, device.ID)
	if err != nil {
		return nil, ErrInvalidCode
	}

	if _, err := s.DeviceVerify.ConsumeByUserDeviceAndCode(ctx, pending.UserID, device.ID, HashToken(code)); err != nil {
		uid := pending.UserID
		if locked, _ := s.DeviceVerify.IncrementFailedAttempts(ctx, dv.ID, s.Policy.MaxOTPAttempts); locked {
			_ = s.Security.Record(ctx, domain.SecurityEvent{
				UserID:    &uid,
				EventType: domain.EventOTPAttemptLimit,
				Severity:  domain.SeverityWarning,
				Details:   map[string]string{"stage": "device_verify"},
			})
			return nil, ErrChallengeLocked
		}
		_, _ = s.BruteForce.RecordOTPFailure(ctx, "device_verify", dv.ID.Hex(), &uid)
		return nil, ErrInvalidCode
	}

	if err := s.Devices.MarkVerified(ctx, device.ID); err != nil {
		return nil, err
	}
	_ = s.Pending.DeleteByTokenHash(ctx, pending.TokenHash)
	_ = s.BruteForce.ResetOTPFailures(ctx, "device_verify", dv.ID.Hex())

	user, err := s.Users.FindByID(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}
	tokens, err := s.issueAuthTokens(ctx, user, device)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Status: LoginStatusAuthenticated, User: user, Tokens: tokens}, nil
}

func (s *Service) GetOrganization(ctx context.Context, userID, organizationID primitive.ObjectID) (*domain.Organization, domain.OrgRole, error) {
	role, err := s.Tenant.RequireMembership(ctx, userID, organizationID)
	if err != nil {
		return nil, "", err
	}
	org, err := s.Orgs.FindByID(ctx, organizationID)
	if err != nil {
		return nil, "", err
	}
	return org, role, nil
}

func (s *Service) ListWorkspaces(ctx context.Context, userID primitive.ObjectID) ([]WorkspaceEntry, error) {
	members, err := s.Members.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceEntry, 0, len(members))
	for _, m := range members {
		org, err := s.Orgs.FindByID(ctx, m.OrganizationID)
		if err != nil {
			continue
		}
		out = append(out, WorkspaceEntry{
			Organization: org,
			Role:         m.Role,
		})
	}
	return out, nil
}

type WorkspaceEntry struct {
	Organization *domain.Organization
	Role         domain.OrgRole
}
