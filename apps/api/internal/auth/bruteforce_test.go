package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	miniredis "github.com/alicebob/miniredis/v2"
)

type noopSecurity struct{}

func (noopSecurity) Record(context.Context, domain.SecurityEvent) error { return nil }

func TestLoginBruteForceLockout(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	ctx := context.Background()
	rc, err := redis.Connect(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	policy := auth.SecurityPolicy{
		LoginMaxAttempts:     3,
		LoginLockoutDuration: time.Minute,
		MaxOTPAttempts:       5,
	}
	guard := &auth.BruteForceGuard{Redis: rc, Policy: policy, Security: noopSecurity{}}

	email := "user@example.com"
	for i := 0; i < 3; i++ {
		locked, err := guard.RecordLoginFailure(ctx, email, nil)
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 && locked {
			t.Fatal("should not lock before threshold")
		}
		if i == 2 && !locked {
			t.Fatal("expected lock at threshold")
		}
	}
	ok, err := guard.IsLoginLocked(ctx, email)
	if err != nil || !ok {
		t.Fatal("expected account locked")
	}
	if err := guard.ResetLoginFailures(ctx, email); err != nil {
		t.Fatal(err)
	}
	ok, err = guard.IsLoginLocked(ctx, email)
	if err != nil || ok {
		t.Fatal("expected lock cleared after successful auth reset")
	}
}

func TestOTPAttemptLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	ctx := context.Background()
	rc, err := redis.Connect(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	policy := auth.SecurityPolicy{MaxOTPAttempts: 2, PendingAuthTTL: time.Minute}
	guard := &auth.BruteForceGuard{Redis: rc, Policy: policy, Security: noopSecurity{}}

	key := "challenge-key"
	blocked, err := guard.RecordOTPFailure(ctx, "device_verify", key, nil)
	if err != nil || blocked {
		t.Fatal("first failure should not block")
	}
	blocked, err = guard.RecordOTPFailure(ctx, "device_verify", key, nil)
	if err != nil || !blocked {
		t.Fatal("second failure should block")
	}
	ok, err := guard.IsOTPBlocked(ctx, "device_verify", key)
	if err != nil || !ok {
		t.Fatal("expected otp blocked")
	}
}
