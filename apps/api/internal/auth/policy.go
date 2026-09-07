package auth

import "time"

// SecurityPolicy centralizes authentication security limits (FINAL_MASTER_PROMPT v1.0.2).
type SecurityPolicy struct {
	MaxOTPAttempts       int
	LoginMaxAttempts     int
	LoginLockoutDuration time.Duration
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	EmailVerifyTTL       time.Duration
	PendingAuthTTL       time.Duration
	MFASetupTTL          time.Duration
	DeviceVerifyTTL      time.Duration
	LoginEmailOTPTTL     time.Duration
	AccountDeletionTTL   time.Duration
}

func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		MaxOTPAttempts:       5,
		LoginMaxAttempts:     5,
		LoginLockoutDuration: 15 * time.Minute,
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      7 * 24 * time.Hour,
		EmailVerifyTTL:       24 * time.Hour,
		PendingAuthTTL:       10 * time.Minute,
		MFASetupTTL:          30 * time.Minute,
		DeviceVerifyTTL:      15 * time.Minute,
		LoginEmailOTPTTL:     10 * time.Minute,
		AccountDeletionTTL:   15 * time.Minute,
	}
}
