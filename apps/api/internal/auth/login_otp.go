package auth

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/httpx"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LoginRequest struct {
	Email             string
	Password          string
	DeviceFingerprint string
	Platform          domain.DevicePlatform
	AppVersion        string
	TurnstileToken    string
}

func (s *Service) requestClientInfo(ctx context.Context) httpx.ClientInfo {
	return httpx.ClientInfoFrom(ctx)
}

func deviceLabel(platform domain.DevicePlatform, userAgent string) string {
	switch platform {
	case domain.DevicePlatformElectronWin:
		return "Miyuna Desktop — Windows"
	case domain.DevicePlatformElectronMac:
		return "Miyuna Desktop — macOS"
	default:
		if userAgent != "" {
			return "Web — " + truncateUA(userAgent, 48)
		}
		return "Web"
	}
}

func truncateUA(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *Service) verifyTurnstile(ctx context.Context, token string) error {
	if s.Turnstile == nil {
		return nil
	}
	info := s.requestClientInfo(ctx)
	return s.Turnstile.Verify(ctx, token, info.IPAddress)
}

func (s *Service) rejectForeignDesktopSession(ctx context.Context, userID primitive.ObjectID, device *domain.Device) error {
	if !domain.IsElectronPlatform(device.Platform) {
		return nil
	}
	sessions, err := s.Sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.DeviceID == device.ID {
			continue
		}
		other, findErr := s.Devices.FindByID(ctx, userID, sess.DeviceID)
		if findErr != nil {
			continue
		}
		if !domain.IsElectronPlatform(other.Platform) {
			continue
		}
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &userID,
			EventType: domain.EventDesktopSessionBlocked,
			Severity:  domain.SeverityWarning,
			Details: map[string]string{
				"activeDeviceId":  other.ID.Hex(),
				"attemptDeviceId": device.ID.Hex(),
			},
		})
		return ErrDesktopSessionActive
	}
	return nil
}

func (s *Service) replaceSameDeviceSessions(ctx context.Context, device *domain.Device) {
	if s.Sessions == nil || device == nil || device.ID.IsZero() {
		return
	}
	_ = s.Sessions.RevokeByDeviceID(ctx, device.ID)
}

func (s *Service) completeLogin(ctx context.Context, user *domain.User, device *domain.Device) (*LoginResult, error) {
	if err := s.Devices.MarkVerified(ctx, device.ID); err != nil {
		return nil, err
	}
	if err := s.rejectForeignDesktopSession(ctx, user.ID, device); err != nil {
		return nil, err
	}
	s.replaceSameDeviceSessions(ctx, device)
	tokens, err := s.issueAuthTokens(ctx, user, device)
	if err != nil {
		return nil, err
	}
	uid := user.ID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventLoginSuccess,
		Severity:  domain.SeverityInfo,
		Details: map[string]string{
			"deviceId": device.ID.Hex(),
			"platform": string(device.Platform),
		},
	})
	if s.Activity != nil {
		did := device.ID
		s.Activity.Record(ctx, user.ID, "LOGIN", &did, &user.PersonalOrgID, "")
	}
	info := s.requestClientInfo(ctx)
	if device.Label != "" && info.IPAddress != "" {
		subject, plain, html := mail.DeviceLoginNoticeEmail(device.Label, info.IPAddress)
		_ = s.Mail.Send(ctx, mail.Message{To: user.Email, Subject: subject, Body: plain, HTMLBody: html})
	}
	return &LoginResult{Status: LoginStatusAuthenticated, User: user, Tokens: tokens}, nil
}

func (s *Service) sendLoginEmailOTP(ctx context.Context, user *domain.User, pendingHash string) error {
	codePlain, codeHash, err := NewNumericCode()
	if err != nil {
		return err
	}
	_ = s.LoginEmailVerify.InvalidateByPendingHash(ctx, pendingHash)
	lev := &domain.LoginEmailVerification{
		UserID:      user.ID,
		PendingHash: pendingHash,
		CodeHash:    codeHash,
		ExpiresAt:   time.Now().UTC().Add(s.Policy.LoginEmailOTPTTL),
	}
	if err := s.LoginEmailVerify.Create(ctx, lev); err != nil {
		return err
	}
	subject, plain, html := mail.LoginOTPEmail(codePlain)
	if err := s.deliverCriticalMail(ctx, mail.Message{To: user.Email, Subject: subject, Body: plain, HTMLBody: html}); err != nil {
		return err
	}
	uid := user.ID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventLoginEmailOTPSent,
		Severity:  domain.SeverityInfo,
		Details:   map[string]string{"stage": "login"},
	})
	return nil
}

func (s *Service) beginEmailOTPLogin(ctx context.Context, user *domain.User, req LoginRequest) (*LoginResult, error) {
	info := s.requestClientInfo(ctx)
	platform := req.Platform
	if platform == "" {
		platform = domain.DevicePlatformWeb
	}
	device := &domain.Device{
		UserID:            user.ID,
		DeviceFingerprint: req.DeviceFingerprint,
		Platform:          platform,
		Label:             deviceLabel(platform, info.UserAgent),
		UserAgent:         info.UserAgent,
		IPAddress:         info.IPAddress,
		AppVersion:        req.AppVersion,
		Verified:          false,
	}
	if err := s.Devices.Upsert(ctx, device); err != nil {
		return nil, err
	}
	if err := s.rejectForeignDesktopSession(ctx, user.ID, device); err != nil {
		return nil, err
	}

	pendingToken, pendingHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	pending := &domain.PendingAuth{
		TokenHash:         pendingHash,
		UserID:            user.ID,
		DeviceFingerprint: req.DeviceFingerprint,
		Stage:             StageEmailOTP,
		ExpiresAt:         time.Now().UTC().Add(s.Policy.PendingAuthTTL),
	}
	if err := s.Pending.Create(ctx, pending); err != nil {
		return nil, err
	}
	if err := s.sendLoginEmailOTP(ctx, user, pendingHash); err != nil {
		return nil, err
	}
	return &LoginResult{
		Status: LoginStatusEmailOTPRequired,
		User:   user,
		Pending: &PendingInfo{
			Token:             pendingToken,
			UserID:            user.ID,
			DeviceFingerprint: req.DeviceFingerprint,
			Stage:             StageEmailOTP,
			ExpiresAt:         pending.ExpiresAt,
		},
	}, nil
}

func (s *Service) VerifyLoginEmailOTP(ctx context.Context, pendingToken, code, deviceFingerprint string) (*LoginResult, error) {
	hash := HashToken(pendingToken)
	pending, err := s.Pending.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if pending.DeviceFingerprint != deviceFingerprint || pending.Stage != StageEmailOTP {
		return nil, ErrInvalidToken
	}

	lev, err := s.LoginEmailVerify.FindActiveByPendingHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidCode
	}

	if _, err := s.LoginEmailVerify.ConsumeByPendingHashAndCode(ctx, hash, HashToken(code)); err != nil {
		uid := pending.UserID
		if locked, _ := s.LoginEmailVerify.IncrementFailedAttempts(ctx, lev.ID, s.Policy.MaxOTPAttempts); locked {
			_ = s.Security.Record(ctx, domain.SecurityEvent{
				UserID:    &uid,
				EventType: domain.EventOTPAttemptLimit,
				Severity:  domain.SeverityWarning,
				Details:   map[string]string{"stage": "login_email_otp"},
			})
			return nil, ErrChallengeLocked
		}
		_, _ = s.BruteForce.RecordOTPFailure(ctx, "login_email_otp", hash, &uid)
		return nil, ErrInvalidCode
	}
	_ = s.BruteForce.ResetOTPFailures(ctx, "login_email_otp", hash)
	_ = s.Pending.DeleteByTokenHash(ctx, pending.TokenHash)

	user, err := s.Users.FindByID(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}
	device, err := s.Devices.FindByUserAndFingerprint(ctx, user.ID, deviceFingerprint)
	if err != nil {
		return nil, err
	}

	if !user.MFAEnabled {
		return s.completeLogin(ctx, user, device)
	}

	newPendingToken, pendingHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	next := &domain.PendingAuth{
		TokenHash:         pendingHash,
		UserID:            user.ID,
		DeviceFingerprint: deviceFingerprint,
		Stage:             StageMFA,
		ExpiresAt:         time.Now().UTC().Add(s.Policy.PendingAuthTTL),
	}
	if err := s.Pending.Create(ctx, next); err != nil {
		return nil, err
	}
	return &LoginResult{
		Status: LoginStatusMFARequired,
		User:   user,
		Pending: &PendingInfo{
			Token:             newPendingToken,
			UserID:            user.ID,
			DeviceFingerprint: deviceFingerprint,
			Stage:             StageMFA,
			ExpiresAt:         next.ExpiresAt,
		},
	}, nil
}

func (s *Service) ResendLoginEmailOTP(ctx context.Context, pendingToken string) (*LoginResult, error) {
	hash := HashToken(pendingToken)
	pending, err := s.Pending.FindByTokenHash(ctx, hash)
	if err != nil || pending.Stage != StageEmailOTP {
		return nil, ErrInvalidToken
	}
	if blocked, _ := s.BruteForce.IsOTPBlocked(ctx, "login_email_resend", hash); blocked {
		return nil, ErrChallengeLocked
	}
	user, err := s.Users.FindByID(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.sendLoginEmailOTP(ctx, user, hash); err != nil {
		return nil, err
	}
	_, _ = s.BruteForce.RecordOTPFailure(ctx, "login_email_resend", hash, &user.ID)
	return &LoginResult{
		Status: LoginStatusEmailOTPRequired,
		User:   user,
		Pending: &PendingInfo{
			Token:             pendingToken,
			UserID:            user.ID,
			DeviceFingerprint: pending.DeviceFingerprint,
			Stage:             StageEmailOTP,
			ExpiresAt:         pending.ExpiresAt,
		},
	}, nil
}

func (s *Service) ListDevices(ctx context.Context, userID primitive.ObjectID) ([]domain.Device, error) {
	return s.Devices.ListByUser(ctx, userID)
}

func (s *Service) ListActivityLog(ctx context.Context, userID primitive.ObjectID, limit int, cursor string) ([]domain.UserActivityLog, error) {
	if s.Activity == nil || s.Activity.Logs == nil {
		return nil, nil
	}
	var before *time.Time
	if cursor != "" {
		if ts, err := time.Parse(time.RFC3339Nano, cursor); err == nil {
			before = &ts
		}
	}
	return s.Activity.Logs.ListByUser(ctx, userID, limit, before)
}

func (s *Service) TurnstileVerify(ctx context.Context, token string) error {
	return s.verifyTurnstile(ctx, token)
}

func (s *Service) RevokeDevice(ctx context.Context, userID, deviceID primitive.ObjectID) error {
	if _, err := s.Devices.FindByID(ctx, userID, deviceID); err != nil {
		return err
	}
	if err := s.Devices.Revoke(ctx, userID, deviceID); err != nil {
		return err
	}
	_ = s.Sessions.RevokeByDeviceID(ctx, deviceID)
	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventDeviceRevoked,
		Severity:  domain.SeverityWarning,
		Details:   map[string]string{"deviceId": deviceID.Hex()},
	})
	if s.Activity != nil {
		s.Activity.Record(ctx, userID, "DEVICE_REVOKED", &deviceID, nil, deviceID.Hex())
	}
	return nil
}
