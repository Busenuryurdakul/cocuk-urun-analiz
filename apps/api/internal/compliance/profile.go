package compliance

import (
	"errors"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var (
	ErrInvalidProfile      = errors.New("invalid compliance profile")
	ErrComplianceViolation = errors.New("compliance violation")
	ErrBypassAttempt       = errors.New("compliance bypass attempt")
	ErrConsentRequired     = errors.New("consent required")
)

var validProfiles = map[domain.ComplianceProfile]bool{
	domain.ProfileKVKK: true,
	domain.ProfileGDPR: true,
	domain.ProfileBoth: true,
}

func ValidateProfile(raw string) (domain.ComplianceProfile, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "OFF" || normalized == "DISABLE" || normalized == "NONE" {
		return "", ErrBypassAttempt
	}
	profile := domain.ComplianceProfile(normalized)
	if !validProfiles[profile] {
		return "", ErrInvalidProfile
	}
	return profile, nil
}

func ProfileAlwaysOn(profile domain.ComplianceProfile) bool {
	return validProfiles[profile]
}
