package compliance

import (
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestComplianceReflexVersion(t *testing.T) {
	if ComplianceReflexVersion() != "1.0.0" {
		t.Fatalf("unexpected reflex version %q", ComplianceReflexVersion())
	}
}

func TestComplianceReflexInstructionProfilesDiffer(t *testing.T) {
	kvkk := ComplianceReflexInstruction(domain.ProfileKVKK, "1.0.0")
	gdpr := ComplianceReflexInstruction(domain.ProfileGDPR, "1.0.0")
	both := ComplianceReflexInstruction(domain.ProfileBoth, "1.0.0")

	if kvkk == gdpr || kvkk == both || gdpr == both {
		t.Fatal("expected distinct reflex instructions per profile")
	}
	if !strings.Contains(kvkk, "KVKK") {
		t.Fatal("expected KVKK marker in KVKK reflex")
	}
	if !strings.Contains(gdpr, "GDPR") {
		t.Fatal("expected GDPR marker in GDPR reflex")
	}
	if !strings.Contains(both, "KVKK") || !strings.Contains(both, "GDPR") {
		t.Fatal("expected both markers in BOTH reflex")
	}
}

func TestComplianceReflexInstructionUnknownVersionFallsBack(t *testing.T) {
	instruction := ComplianceReflexInstruction(domain.ProfileGDPR, "9.9.9")
	if !strings.Contains(instruction, "9.9.9 unavailable") {
		t.Fatal("expected unknown version notice")
	}
	if !strings.Contains(instruction, "GDPR scope") {
		t.Fatal("expected GDPR reflex body after fallback notice")
	}
}
