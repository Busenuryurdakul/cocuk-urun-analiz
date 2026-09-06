package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestElectronSecondSessionBlockedSameDeviceRehydrates(t *testing.T) {
	svc, cleanup := setupAuthService(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()
	orgID := primitive.NewObjectID()
	user := &domain.User{ID: userID, PersonalOrgID: orgID, EmailVerified: true, MFAEnabled: true}

	first := &domain.Device{
		UserID:            userID,
		DeviceFingerprint: "desktop-a",
		Platform:          domain.DevicePlatformElectronWin,
		Verified:          true,
	}
	if err := svc.Devices.Upsert(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.issueAuthTokens(ctx, user, first); err != nil {
		t.Fatal(err)
	}

	second := &domain.Device{
		UserID:            userID,
		DeviceFingerprint: "desktop-b",
		Platform:          domain.DevicePlatformElectronMac,
		Verified:          true,
	}
	if err := svc.Devices.Upsert(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := svc.rejectForeignDesktopSession(ctx, userID, second); !errors.Is(err, ErrDesktopSessionActive) {
		t.Fatalf("expected DESKTOP_SESSION_ACTIVE, got %v", err)
	}

	if err := svc.rejectForeignDesktopSession(ctx, userID, first); err != nil {
		t.Fatalf("same device must rehydrate: %v", err)
	}

	web := &domain.Device{
		UserID:            userID,
		DeviceFingerprint: "web-browser",
		Platform:          domain.DevicePlatformWeb,
		Verified:          true,
	}
	if err := svc.Devices.Upsert(ctx, web); err != nil {
		t.Fatal(err)
	}
	if err := svc.rejectForeignDesktopSession(ctx, userID, web); err != nil {
		t.Fatalf("web session must stay allowed: %v", err)
	}

	svc.replaceSameDeviceSessions(ctx, first)
	if _, err := svc.issueAuthTokens(ctx, user, first); err != nil {
		t.Fatal(err)
	}
	sessions, err := svc.Sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	sameDevice := 0
	for _, sess := range sessions {
		if sess.DeviceID == first.ID {
			sameDevice++
		}
	}
	if sameDevice != 1 {
		t.Fatalf("same device should keep a single active session, got %d", sameDevice)
	}
}
