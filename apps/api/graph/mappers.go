package graph

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/httpx"
	orgsvc "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/org"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	errUnauthorized = errors.New("unauthorized")
	errForbidden    = errors.New("forbidden")
)

func gqlError(code string, err error) error {
	return &gqlerror.Error{
		Message: err.Error(),
		Extensions: map[string]any{
			"code": code,
		},
	}
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return gqlError("INVALID_CREDENTIALS", err)
	case errors.Is(err, auth.ErrEmailNotVerified):
		return gqlError("EMAIL_NOT_VERIFIED", err)
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInvalidCode):
		return gqlError("INVALID_TOKEN", err)
	case errors.Is(err, auth.ErrChallengeLocked), errors.Is(err, auth.ErrAccountLocked):
		return gqlError("CHALLENGE_LOCKED", err)
	case errors.Is(err, auth.ErrRefreshReplay):
		return gqlError("REFRESH_REPLAY", err)
	case errors.Is(err, auth.ErrDesktopSessionActive):
		return gqlError("DESKTOP_SESSION_ACTIVE", err)
	case errors.Is(err, tenant.ErrCrossTenantAccess), errors.Is(err, orgsvc.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, orgsvc.ErrLastOwner):
		return gqlError("LAST_OWNER", err)
	case errors.Is(err, orgsvc.ErrPersonalOrg):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, orgsvc.ErrInvalidRole):
		return gqlError("INVALID_ROLE", err)
	case errors.Is(err, orgsvc.ErrInvitationExpired):
		return gqlError("INVITATION_EXPIRED", err)
	case errors.Is(err, orgsvc.ErrInvitationMismatch):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, orgsvc.ErrInvitationUsed):
		return gqlError("INVALID_TOKEN", err)
	case errors.Is(err, compliance.ErrInvalidProfile):
		return gqlError("INVALID_PROFILE", err)
	case errors.Is(err, compliance.ErrInvalidPolicyVersion):
		return gqlError("INVALID_PROFILE", err)
	case errors.Is(err, compliance.ErrComplianceViolation):
		return gqlError("COMPLIANCE_VIOLATION", err)
	case errors.Is(err, compliance.ErrBypassAttempt):
		return gqlError("COMPLIANCE_VIOLATION", err)
	case errors.Is(err, compliance.ErrConsentRequired):
		return gqlError("CONSENT_REQUIRED", err)
	case errors.Is(err, repository.ErrDuplicate):
		return gqlError("DUPLICATE", err)
	default:
		return err
	}
}

func responseWriter(ctx context.Context) (http.ResponseWriter, bool) {
	return httpx.ResponseWriterFrom(ctx)
}

func requireSession(ctx context.Context) (httpx.SessionContext, error) {
	session, ok := httpx.SessionFrom(ctx)
	if !ok || session.UserID == "" {
		return httpx.SessionContext{}, errUnauthorized
	}
	return session, nil
}

func toModelUser(u *domain.User) *model.User {
	return &model.User{
		ID:            u.ID.Hex(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		MfaEnabled:    u.MFAEnabled,
		PersonalOrgID: u.PersonalOrgID.Hex(),
	}
}

func toModelLoginStatus(status auth.LoginStatus) model.LoginStatus {
	switch status {
	case auth.LoginStatusEmailOTPRequired:
		return model.LoginStatusEmailOtpRequired
	case auth.LoginStatusMFASetupRequired:
		return model.LoginStatusMfaSetupRequired
	case auth.LoginStatusMFARequired:
		return model.LoginStatusMfaRequired
	case auth.LoginStatusDeviceVerificationRequired:
		return model.LoginStatusDeviceVerificationRequired
	default:
		return model.LoginStatusAuthenticated
	}
}

func toModelLoginPayload(result *auth.LoginResult) *model.LoginPayload {
	payload := &model.LoginPayload{
		Status: toModelLoginStatus(result.Status),
	}
	if result.User != nil {
		payload.User = toModelUser(result.User)
	}
	if result.MFASetup != nil {
		payload.MfaSetup = &model.MFASetupPayload{
			Secret:     result.MFASetup.Secret,
			OtpauthURL: result.MFASetup.OTPAuthURL,
		}
	}
	return payload
}

func applyAuthCookies(ctx context.Context, opts cookies.Options, tokens *auth.AuthTokens) {
	w, ok := responseWriter(ctx)
	if !ok || tokens == nil {
		return
	}
	if tokens.AccessToken != "" {
		cookies.Set(w, cookies.AccessCookie, tokens.AccessToken, tokens.AccessExpiresAt, opts)
	}
	if tokens.RefreshToken != "" {
		cookies.Set(w, cookies.RefreshCookie, tokens.RefreshToken, tokens.RefreshExpiresAt, opts)
	}
}

func clearAuthCookies(ctx context.Context, opts cookies.Options) {
	w, ok := responseWriter(ctx)
	if !ok {
		return
	}
	cookies.Clear(w, cookies.AccessCookie, opts)
	cookies.Clear(w, cookies.RefreshCookie, opts)
	cookies.Clear(w, cookies.PendingCookie, opts)
	cookies.Clear(w, cookies.SetupCookie, opts)
}

func applyLoginCookies(ctx context.Context, opts cookies.Options, result *auth.LoginResult) {
	w, ok := responseWriter(ctx)
	if !ok || result == nil {
		return
	}
	switch result.Status {
	case auth.LoginStatusAuthenticated:
		applyAuthCookies(ctx, opts, result.Tokens)
		cookies.Clear(w, cookies.PendingCookie, opts)
		cookies.Clear(w, cookies.SetupCookie, opts)
	case auth.LoginStatusMFARequired, auth.LoginStatusDeviceVerificationRequired, auth.LoginStatusEmailOTPRequired:
		if result.Pending != nil {
			cookies.Set(w, cookies.PendingCookie, result.Pending.Token, result.Pending.ExpiresAt, opts)
		}
	case auth.LoginStatusMFASetupRequired:
		if result.MFASetup != nil {
			cookies.Set(w, cookies.SetupCookie, result.MFASetup.Token, result.MFASetup.ExpiresAt, opts)
		}
	}
}

func applyMFASetupCookie(ctx context.Context, opts cookies.Options, setup *auth.MFASetupInfo) {
	w, ok := responseWriter(ctx)
	if !ok || setup == nil {
		return
	}
	cookies.Set(w, cookies.SetupCookie, setup.Token, setup.ExpiresAt, opts)
}

func parseObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

func ptrStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func toDomainPlatform(p *model.ClientPlatform) domain.DevicePlatform {
	if p == nil {
		return domain.DevicePlatformWeb
	}
	switch *p {
	case model.ClientPlatformElectronWin:
		return domain.DevicePlatformElectronWin
	case model.ClientPlatformElectronMac:
		return domain.DevicePlatformElectronMac
	default:
		return domain.DevicePlatformWeb
	}
}

func toModelDevice(d domain.Device) *model.Device {
	return &model.Device{
		ID:           d.ID.Hex(),
		Platform:     toModelPlatform(d.Platform),
		Label:        d.Label,
		UserAgent:    strPtr(d.UserAgent),
		IPAddress:    strPtr(d.IPAddress),
		AppVersion:   strPtr(d.AppVersion),
		Verified:     d.Verified,
		LastActiveAt: d.LastActiveAt.Format(time.RFC3339),
		CreatedAt:    d.CreatedAt.Format(time.RFC3339),
	}
}

func toModelPlatform(p domain.DevicePlatform) model.ClientPlatform {
	switch p {
	case domain.DevicePlatformElectronWin:
		return model.ClientPlatformElectronWin
	case domain.DevicePlatformElectronMac:
		return model.ClientPlatformElectronMac
	default:
		return model.ClientPlatformWeb
	}
}

func toModelActivityLog(e domain.UserActivityLog) *model.ActivityLogEntry {
	return &model.ActivityLogEntry{
		ID:         e.ID.Hex(),
		Action:     e.Action,
		ResourceID: strPtr(e.ResourceID),
		IPAddress:  strPtr(e.IPAddress),
		UserAgent:  strPtr(e.UserAgent),
		Timestamp:  e.Timestamp.Format(time.RFC3339),
	}
}

func toModelOrgType(t domain.OrgType) model.OrgType {
	if t == domain.OrgTypeOrganization {
		return model.OrgTypeOrganization
	}
	return model.OrgTypePersonal
}

func toModelOrgRole(r domain.OrgRole) model.OrgRole {
	return model.OrgRole(r)
}

func toModelOrganization(o *domain.Organization) *model.Organization {
	return &model.Organization{
		ID:                      o.ID.Hex(),
		Name:                    o.Name,
		Type:                    toModelOrgType(o.Type),
		ComplianceProfile:       model.ComplianceProfile(o.ComplianceProfile),
		CompliancePolicyVersion: o.CompliancePolicyVersion,
	}
}

func toModelCompliancePolicy(p *domain.CompliancePolicyVersion) *model.CompliancePolicy {
	return &model.CompliancePolicy{
		Profile:     model.ComplianceProfile(p.Profile),
		Version:     p.Version,
		Status:      model.PolicyStatus(p.Status),
		EffectiveAt: p.EffectiveAt.UTC().Format(time.RFC3339),
		Reason:      &p.Reason,
	}
}

func toModelConsent(c *domain.Consent) *model.Consent {
	out := &model.Consent{
		ID:            c.ID.Hex(),
		Purpose:       model.ConsentPurpose(c.Purpose),
		PolicyVersion: c.PolicyVersion,
		GrantedAt:     c.GrantedAt.UTC().Format(time.RFC3339),
	}
	if c.OrganizationID != nil {
		id := c.OrganizationID.Hex()
		out.OrganizationID = &id
	}
	if c.WithdrawnAt != nil {
		w := c.WithdrawnAt.UTC().Format(time.RFC3339)
		out.WithdrawnAt = &w
	}
	return out
}

func requireActor(ctx context.Context) (primitive.ObjectID, error) {
	session, err := requireSession(ctx)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return parseObjectID(session.UserID)
}
