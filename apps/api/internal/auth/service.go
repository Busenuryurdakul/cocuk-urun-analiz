package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
)

const (
	StageMFA        = "MFA"
	StageDevice     = "DEVICE"
	sessionTTL      = 24 * time.Hour
	pendingTTL      = 10 * time.Minute
	emailVerifyTTL  = 24 * time.Hour
	mfaSetupTTL     = 30 * time.Minute
	deviceVerifyTTL = 15 * time.Minute
)

type SessionInfo struct {
	Token          string
	UserID         primitive.ObjectID
	OrganizationID primitive.ObjectID
	DeviceID       primitive.ObjectID
	ExpiresAt      time.Time
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
	Session  *SessionInfo
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
	Pending      *repository.PendingAuthRepository
	EmailVerify  *repository.EmailVerificationRepository
	DeviceVerify *repository.DeviceVerificationRepository
	MFASetup     *repository.MFASetupRepository
	Security     *repository.SecurityEventRepository
	Mail         mail.Service
	Tenant       *tenant.Guard
	WebBaseURL   string
	MFAIssuer    string
}

func (s *Service) Register(ctx context.Context, email, password string) (primitive.ObjectID, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 {
		return primitive.NilObjectID, ErrInvalidCredentials
	}
	hash, err := HashPassword(password)
	if err != nil {
		return primitive.NilObjectID, err
	}

	user := &domain.User{
		Email:         email,
		PasswordHash:  hash,
		EmailVerified: false,
		MFAEnabled:    false,
	}
	if err := s.Users.Create(ctx, user); err != nil {
		return primitive.NilObjectID, err
	}

	org := &domain.Organization{
		Name:                    "Personal Workspace",
		Type:                    domain.OrgTypePersonal,
		ComplianceProfile:       "KVKK",
		CompliancePolicyVersion: "v1",
		OwnerID:                 user.ID,
	}
	if err := s.Orgs.Create(ctx, org); err != nil {
		return primitive.NilObjectID, err
	}

	user.PersonalOrgID = org.ID
	if err := s.Users.Update(ctx, user); err != nil {
		return primitive.NilObjectID, err
	}

	member := &domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           domain.RoleOwner,
	}
	if err := s.Members.Create(ctx, member); err != nil {
		return primitive.NilObjectID, err
	}

	token, tokenHash, err := NewToken()
	if err != nil {
		return primitive.NilObjectID, err
	}
	ev := &domain.EmailVerification{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(emailVerifyTTL),
	}
	if err := s.EmailVerify.Create(ctx, ev); err != nil {
		return primitive.NilObjectID, err
	}

	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", strings.TrimRight(s.WebBaseURL, "/"), token)
	_ = s.Mail.Send(ctx, mail.Message{
		To:      email,
		Subject: "Miyuna — e-posta doğrulama",
		Body:    fmt.Sprintf("E-posta adresinizi doğrulamak için bağlantıyı açın:\n%s", verifyURL),
	})

	return user.ID, nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) (*MFASetupInfo, error) {
	ev, err := s.EmailVerify.ConsumeByTokenHash(ctx, HashToken(token))
	if err != nil {
		return nil, ErrInvalidToken
	}
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
		ExpiresAt: time.Now().UTC().Add(mfaSetupTTL),
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

func (s *Service) GetMFASetup(ctx context.Context, setupToken string) (*MFASetupInfo, error) {
	challenge, err := s.MFASetup.FindByTokenHash(ctx, HashToken(setupToken))
	if err != nil {
		return nil, ErrInvalidToken
	}
	user, err := s.Users.FindByID(ctx, challenge.UserID)
	if err != nil {
		return nil, err
	}
	_, url, err := GenerateTOTPSecret(s.MFAIssuer, user.Email)
	if err != nil {
		return nil, err
	}
	return &MFASetupInfo{
		Token:      setupToken,
		Secret:     challenge.Secret,
		OTPAuthURL: url,
		ExpiresAt:  challenge.ExpiresAt,
	}, nil
}

func (s *Service) ConfirmMFA(ctx context.Context, setupToken, code string) error {
	challenge, err := s.MFASetup.ConsumeByTokenHash(ctx, HashToken(setupToken))
	if err != nil {
		return ErrInvalidToken
	}
	if !ValidateTOTP(challenge.Secret, code) {
		uid := challenge.UserID
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventMFAFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"stage": "mfa_setup"},
		})
		return ErrInvalidCode
	}
	return s.Users.EnableMFA(ctx, challenge.UserID, challenge.Secret)
}

func (s *Service) Login(ctx context.Context, email, password, deviceFingerprint string) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !CheckPassword(user.PasswordHash, password) {
		uid := user.ID
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventAuthFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"reason": "bad_password"},
		})
		return nil, ErrInvalidCredentials
	}
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}
	if !user.MFAEnabled {
		setup, err := s.beginMFASetup(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResult{
			Status:   LoginStatusMFASetupRequired,
			User:     user,
			MFASetup: setup,
		}, nil
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
		ExpiresAt:         time.Now().UTC().Add(pendingTTL),
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
	pending, err := s.Pending.FindByTokenHash(ctx, HashToken(pendingToken))
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
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventMFAFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"stage": "login"},
		})
		return nil, ErrInvalidCode
	}
	_ = s.Pending.DeleteByTokenHash(ctx, pending.TokenHash)

	device, err := s.Devices.FindByUserAndFingerprint(ctx, user.ID, deviceFingerprint)
	if err != nil {
		return nil, err
	}
	if device.Verified {
		session, err := s.createSession(ctx, user, device)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Status: LoginStatusAuthenticated, User: user, Session: session}, nil
	}

	codePlain, codeHash, err := NewNumericCode()
	if err != nil {
		return nil, err
	}
	dv := &domain.DeviceVerification{
		UserID:    user.ID,
		DeviceID:  device.ID,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().UTC().Add(deviceVerifyTTL),
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
		ExpiresAt:         time.Now().UTC().Add(pendingTTL),
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
	pending, err := s.Pending.FindByTokenHash(ctx, HashToken(pendingToken))
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
	if _, err := s.DeviceVerify.ConsumeByUserDeviceAndCode(ctx, pending.UserID, device.ID, HashToken(code)); err != nil {
		return nil, ErrInvalidCode
	}
	if err := s.Devices.MarkVerified(ctx, device.ID); err != nil {
		return nil, err
	}
	_ = s.Pending.DeleteByTokenHash(ctx, pending.TokenHash)

	user, err := s.Users.FindByID(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}
	session, err := s.createSession(ctx, user, device)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Status: LoginStatusAuthenticated, User: user, Session: session}, nil
}

func (s *Service) createSession(ctx context.Context, user *domain.User, device *domain.Device) (*SessionInfo, error) {
	token, tokenHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(sessionTTL)
	session := &domain.Session{
		TokenHash:      tokenHash,
		UserID:         user.ID,
		OrganizationID: user.PersonalOrgID,
		DeviceID:       device.ID,
		ExpiresAt:      expires,
	}
	if err := s.Sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return &SessionInfo{
		Token:          token,
		UserID:         user.ID,
		OrganizationID: user.PersonalOrgID,
		DeviceID:       device.ID,
		ExpiresAt:      expires,
	}, nil
}

func (s *Service) SessionFromToken(ctx context.Context, token string) (*SessionInfo, error) {
	session, err := s.Sessions.FindByTokenHash(ctx, HashToken(token))
	if err != nil {
		return nil, ErrInvalidToken
	}
	return &SessionInfo{
		Token:          token,
		UserID:         session.UserID,
		OrganizationID: session.OrganizationID,
		DeviceID:       session.DeviceID,
		ExpiresAt:      session.ExpiresAt,
	}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.Sessions.DeleteByTokenHash(ctx, HashToken(token))
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

func (s *Service) SwitchWorkspace(ctx context.Context, sessionToken string, targetOrgID primitive.ObjectID) (*SessionInfo, error) {
	sessionInfo, err := s.SessionFromToken(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	if _, err := s.Tenant.RequireMembership(ctx, sessionInfo.UserID, targetOrgID); err != nil {
		return nil, err
	}
	if err := s.Sessions.UpdateOrganization(ctx, HashToken(sessionToken), targetOrgID); err != nil {
		return nil, err
	}
	sessionInfo.OrganizationID = targetOrgID
	return sessionInfo, nil
}
