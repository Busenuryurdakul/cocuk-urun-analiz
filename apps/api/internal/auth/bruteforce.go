package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SecurityRecorder interface {
	Record(ctx context.Context, event domain.SecurityEvent) error
}

type BruteForceGuard struct {
	Redis    *redis.Client
	Policy   SecurityPolicy
	Security SecurityRecorder
}

func (g *BruteForceGuard) loginFailKey(email string) string {
	return "miyuna:login:fail:" + strings.ToLower(strings.TrimSpace(email))
}

func (g *BruteForceGuard) loginLockKey(email string) string {
	return "miyuna:login:lock:" + strings.ToLower(strings.TrimSpace(email))
}

func (g *BruteForceGuard) otpFailKey(scope, key string) string {
	return fmt.Sprintf("miyuna:otp:fail:%s:%s", scope, key)
}

func (g *BruteForceGuard) IsLoginLocked(ctx context.Context, email string) (bool, error) {
	return g.Redis.Exists(ctx, g.loginLockKey(email))
}

func (g *BruteForceGuard) RecordLoginFailure(ctx context.Context, email string, userID *primitive.ObjectID) (locked bool, err error) {
	failKey := g.loginFailKey(email)
	count, err := g.Redis.IncrWithTTL(ctx, failKey, g.Policy.LoginLockoutDuration)
	if err != nil {
		return false, err
	}
	if count >= int64(g.Policy.LoginMaxAttempts) {
		_ = g.Redis.Set(ctx, g.loginLockKey(email), "1", g.Policy.LoginLockoutDuration)
		_ = g.Security.Record(ctx, domain.SecurityEvent{
			UserID:    userID,
			EventType: domain.EventAuthFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"reason": "login_lockout"},
		})
		return true, nil
	}
	return false, nil
}

func (g *BruteForceGuard) ResetLoginFailures(ctx context.Context, email string) error {
	return g.Redis.Del(ctx, g.loginFailKey(email), g.loginLockKey(email))
}

func (g *BruteForceGuard) RecordOTPFailure(ctx context.Context, scope, key string, userID *primitive.ObjectID) (blocked bool, err error) {
	failKey := g.otpFailKey(scope, key)
	count, err := g.Redis.IncrWithTTL(ctx, failKey, g.Policy.PendingAuthTTL)
	if err != nil {
		return false, err
	}
	if count >= int64(g.Policy.MaxOTPAttempts) {
		_ = g.Security.Record(ctx, domain.SecurityEvent{
			UserID:    userID,
			EventType: domain.EventMFAFailure,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"reason": "otp_attempt_limit", "scope": scope},
		})
		return true, nil
	}
	return false, nil
}

func (g *BruteForceGuard) ResetOTPFailures(ctx context.Context, scope, key string) error {
	return g.Redis.Del(ctx, g.otpFailKey(scope, key))
}

func (g *BruteForceGuard) IsOTPBlocked(ctx context.Context, scope, key string) (bool, error) {
	count, err := g.Redis.GetInt(ctx, g.otpFailKey(scope, key))
	if err != nil {
		return false, nil
	}
	return count >= int64(g.Policy.MaxOTPAttempts), nil
}
