package graph

import (
	"context"
	"errors"
	"net/http"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/httpx"
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
	case errors.Is(err, tenant.ErrCrossTenantAccess):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, repository.ErrDuplicate):
		return gqlError("DUPLICATE_EMAIL", err)
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
	return payload
}

func applyLoginCookies(ctx context.Context, opts cookies.Options, result *auth.LoginResult) {
	w, ok := responseWriter(ctx)
	if !ok || result == nil {
		return
	}
	switch result.Status {
	case auth.LoginStatusAuthenticated:
		if result.Session != nil {
			cookies.Set(w, cookies.SessionCookie, result.Session.Token, result.Session.ExpiresAt, opts)
			cookies.Clear(w, cookies.PendingCookie, opts)
			cookies.Clear(w, cookies.SetupCookie, opts)
		}
	case auth.LoginStatusMFARequired, auth.LoginStatusDeviceVerificationRequired:
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

func toModelOrgType(t domain.OrgType) model.OrgType {
	if t == domain.OrgTypeOrganization {
		return model.OrgTypeOrganization
	}
	return model.OrgTypePersonal
}

func toModelOrgRole(r domain.OrgRole) model.OrgRole {
	return model.OrgRole(r)
}
