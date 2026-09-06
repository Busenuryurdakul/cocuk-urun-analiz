package auth

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	miniredis "github.com/alicebob/miniredis/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupAuthService(t *testing.T) (*Service, func()) {
	t.Helper()
	uri := "mongodb://localhost:27017/miyuna_test_refresh"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongo, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	_ = mongo.EnsureIndexes(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rc, err := redis.Connect(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}

	db := mongo.DB
	security := repository.NewSecurityEventRepository(db)
	policy := DefaultSecurityPolicy()
	policy.RefreshTokenTTL = time.Hour
	svc := &Service{
		Users:            repository.NewUserRepository(db),
		Orgs:             repository.NewOrganizationRepository(db),
		Members:          repository.NewMemberRepository(db),
		Devices:          repository.NewDeviceRepository(db),
		Sessions:         repository.NewSessionRepository(db),
		Rotated:          repository.NewRotatedRefreshRepository(db),
		Pending:          repository.NewPendingAuthRepository(db),
		EmailVerify:      repository.NewEmailVerificationRepository(db),
		LoginEmailVerify: repository.NewLoginEmailVerificationRepository(db),
		DeviceVerify:     repository.NewDeviceVerificationRepository(db),
		MFASetup:         repository.NewMFASetupRepository(db),
		Security:         security,
		BruteForce:       &BruteForceGuard{Redis: rc, Policy: policy, Security: security},
		JWT:              NewJWTManager("test-jwt-secret-key-32chars!", policy.AccessTokenTTL),
		Policy:           policy,
		WebBaseURL:       "http://localhost:3000",
		MFAIssuer:        "Miyuna",
	}

	cleanup := func() {
		_ = mongo.Disconnect(context.Background())
		rc.Close()
		mr.Close()
	}
	return svc, cleanup
}

func TestRefreshRotationAndReplay(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()
	orgID := primitive.NewObjectID()
	deviceID := primitive.NewObjectID()

	user := &domain.User{ID: userID, Email: "refresh@example.com", PersonalOrgID: orgID, EmailVerified: true, MFAEnabled: true}
	device := &domain.Device{ID: deviceID, UserID: userID, DeviceFingerprint: "fp", Platform: domain.DevicePlatformWeb, Verified: true}

	tokens, err := svc.issueAuthTokens(ctx, user, device)
	if err != nil {
		t.Fatal(err)
	}
	oldRefresh := tokens.RefreshToken

	rotated, err := svc.RefreshTokens(ctx, oldRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == oldRefresh {
		t.Fatal("refresh token must rotate")
	}

	_, err = svc.RefreshTokens(ctx, oldRefresh)
	if err == nil {
		t.Fatal("replayed refresh token must be rejected")
	}
}

func TestJWTAccessTokenValidation(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()
	orgID := primitive.NewObjectID()
	deviceID := primitive.NewObjectID()
	user := &domain.User{ID: userID, PersonalOrgID: orgID}
	device := &domain.Device{ID: deviceID, UserID: userID, Verified: true}

	tokens, err := svc.issueAuthTokens(ctx, user, device)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := svc.ValidateAccessToken(ctx, tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.UserID != userID {
		t.Fatal("user id mismatch")
	}
}
