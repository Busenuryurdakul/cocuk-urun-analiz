package auth

import (
	"context"
	"errors"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrRefreshReplay = errors.New("refresh token replay detected")
var ErrAccountLocked = errors.New("account temporarily locked")

type AuthTokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	SessionID        primitive.ObjectID
	UserID           primitive.ObjectID
	OrganizationID   primitive.ObjectID
	DeviceID         primitive.ObjectID
}

func (s *Service) issueAuthTokens(ctx context.Context, user *domain.User, device *domain.Device) (*AuthTokens, error) {
	refreshRaw, refreshHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	familyID := primitive.NewObjectID()
	refreshExpires := time.Now().UTC().Add(s.Policy.RefreshTokenTTL)
	session := &domain.Session{
		UserID:           user.ID,
		OrganizationID:   user.PersonalOrgID,
		DeviceID:         device.ID,
		FamilyID:         familyID,
		RefreshTokenHash: refreshHash,
		Revoked:          false,
		ExpiresAt:        refreshExpires,
	}
	if err := s.Sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	access, accessExp, err := s.JWT.IssueAccessToken(user.ID, session.ID, user.PersonalOrgID)
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     refreshRaw,
		RefreshExpiresAt: refreshExpires,
		SessionID:        session.ID,
		UserID:           user.ID,
		OrganizationID:   user.PersonalOrgID,
		DeviceID:         device.ID,
	}, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	hash := HashToken(refreshToken)
	session, err := s.Sessions.FindByRefreshHash(ctx, hash)
	if err == nil {
		return s.rotateRefresh(ctx, session, hash, refreshToken)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	rotated, err := s.Rotated.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	uid := rotated.SessionID
	_ = s.Sessions.RevokeFamily(ctx, rotated.FamilyID)
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    nil,
		EventType: domain.EventRefreshTokenReplay,
		Severity:  domain.SeverityCritical,
		Details:   map[string]string{"sessionId": uid.Hex(), "familyId": rotated.FamilyID.Hex()},
	})
	return nil, ErrRefreshReplay
}

func (s *Service) rotateRefresh(ctx context.Context, session *domain.Session, oldHash, _ string) (*AuthTokens, error) {
	newRefresh, newHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	refreshExpires := time.Now().UTC().Add(s.Policy.RefreshTokenTTL)
	if err := s.Rotated.Record(ctx, &domain.RotatedRefreshToken{
		SessionID: session.ID,
		FamilyID:  session.FamilyID,
		TokenHash: oldHash,
		ExpiresAt: refreshExpires,
	}); err != nil {
		return nil, err
	}
	if err := s.Sessions.RotateRefresh(ctx, session.ID, newHash, refreshExpires); err != nil {
		return nil, err
	}
	access, accessExp, err := s.JWT.IssueAccessToken(session.UserID, session.ID, session.OrganizationID)
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     newRefresh,
		RefreshExpiresAt: refreshExpires,
		SessionID:        session.ID,
		UserID:           session.UserID,
		OrganizationID:   session.OrganizationID,
		DeviceID:         session.DeviceID,
	}, nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (*AuthTokens, error) {
	claims, err := s.JWT.ParseAccessToken(accessToken)
	if err != nil {
		return nil, err
	}
	sessionID, err := primitive.ObjectIDFromHex(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}
	session, err := s.Sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)
	orgID, _ := primitive.ObjectIDFromHex(claims.OrganizationID)
	return &AuthTokens{
		AccessToken:      accessToken,
		AccessExpiresAt:  claims.ExpiresAt.Time,
		SessionID:        session.ID,
		UserID:           userID,
		OrganizationID:   orgID,
		DeviceID:         session.DeviceID,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}

func (s *Service) LogoutSession(ctx context.Context, sessionID primitive.ObjectID) error {
	return s.Sessions.RevokeByID(ctx, sessionID)
}

func (s *Service) SwitchWorkspaceTokens(ctx context.Context, sessionID, targetOrgID primitive.ObjectID) (*AuthTokens, error) {
	session, err := s.Sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Tenant.RequireMembership(ctx, session.UserID, targetOrgID); err != nil {
		return nil, err
	}
	if err := s.Sessions.UpdateOrganization(ctx, sessionID, targetOrgID); err != nil {
		return nil, err
	}
	access, accessExp, err := s.JWT.IssueAccessToken(session.UserID, session.ID, targetOrgID)
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		SessionID:        session.ID,
		UserID:           session.UserID,
		OrganizationID:   targetOrgID,
		DeviceID:         session.DeviceID,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}
