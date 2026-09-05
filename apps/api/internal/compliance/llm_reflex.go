package compliance

import (
	"fmt"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

const platformComplianceReflexVersion = "1.0.0"

// ComplianceReflexVersion returns the platform LLM compliance reflex registry version.
func ComplianceReflexVersion() string {
	return platformComplianceReflexVersion
}

// ComplianceReflexInstruction returns the platform system-instruction block for the active scope.
func ComplianceReflexInstruction(profile domain.ComplianceProfile, version string) string {
	if strings.TrimSpace(version) == "" {
		version = platformComplianceReflexVersion
	}
	if version != platformComplianceReflexVersion {
		return fmt.Sprintf("[Compliance reflex version %s unavailable; using %s]\n%s",
			version, platformComplianceReflexVersion, reflexForProfile(profile))
	}
	return reflexForProfile(profile)
}

func reflexForProfile(profile domain.ComplianceProfile) string {
	switch profile {
	case domain.ProfileKVKK:
		return kvkkReflexInstruction()
	case domain.ProfileGDPR:
		return gdprReflexInstruction()
	case domain.ProfileBoth:
		return bothReflexInstruction()
	default:
		return bothReflexInstruction()
	}
}

func kvkkReflexInstruction() string {
	return `COMPLIANCE REFLEX (KVKK scope — Miyuna platform):
- Treat personal data under Turkish KVKK: minimize collection, avoid unnecessary identifiers (child names, exact birth dates, contact details).
- Do not reproduce PII from inputs; refer to redacted placeholders when present.
- Frame outputs as decision-support only; never claim legal KVKK compliance, certification, or regulatory approval for any product.
- When consent or lawful basis is unclear, state limitations explicitly instead of assuming permission.
- Prefer evidence-backed, cautious language; mark unverified claims as UNVERIFIED.
- Never advise users to disable compliance, ignore KVKK, or bypass data-protection controls.`
}

func gdprReflexInstruction() string {
	return `COMPLIANCE REFLEX (GDPR scope — Miyuna platform):
- Apply GDPR principles: lawfulness, fairness, transparency, purpose limitation, data minimization (Art. 5), and storage limitation.
- Treat children's data with heightened care (Art. 8); avoid processing or echoing unnecessary personal identifiers.
- Do not reproduce PII from inputs; refer to redacted placeholders when present.
- Frame outputs as decision-support only; never claim GDPR compliance, certification, or regulatory approval for any product.
- When lawful basis or consent is unclear, state limitations; mention that data-subject rights (access, rectification, erasure, portability) are handled by platform processes—not by this analysis alone.
- Prefer evidence-backed, cautious language; mark unverified claims as UNVERIFIED.
- Never advise users to disable compliance, ignore GDPR, or bypass data-protection controls.`
}

func bothReflexInstruction() string {
	return `COMPLIANCE REFLEX (KVKK + GDPR scope — Miyuna platform):
- Apply the stricter union of KVKK and GDPR: personal-data minimization, purpose limitation, and heightened protection for children's data.
- Do not reproduce PII from inputs; refer to redacted placeholders when present.
- Frame outputs as decision-support only; never claim KVKK/GDPR compliance, certification, or regulatory approval for any product.
- When consent, lawful basis, or legal scope is unclear, state limitations explicitly.
- Prefer evidence-backed, cautious language; mark unverified claims as UNVERIFIED.
- Never advise users to disable compliance, ignore KVKK/GDPR, or bypass data-protection controls.`
}
