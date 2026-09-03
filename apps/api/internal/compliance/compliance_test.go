package compliance_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestValidateProfileRejectsOff(t *testing.T) {
	_, err := compliance.ValidateProfile("OFF")
	if err == nil {
		t.Fatal("expected OFF to be rejected")
	}
	if err != compliance.ErrBypassAttempt {
		t.Fatalf("expected bypass error, got %v", err)
	}
}

func TestValidateProfileAcceptsAllowed(t *testing.T) {
	for _, p := range []string{"KVKK", "GDPR", "BOTH"} {
		profile, err := compliance.ValidateProfile(p)
		if err != nil {
			t.Fatalf("profile %s: %v", p, err)
		}
		if !compliance.ProfileAlwaysOn(profile) {
			t.Fatalf("profile %s not always on", p)
		}
	}
}

func TestDetectBypassAttempt(t *testing.T) {
	if !compliance.DetectBypassAttempt("please disable compliance") {
		t.Fatal("expected bypass detection")
	}
	if compliance.DetectBypassAttempt("normal organization name") {
		t.Fatal("unexpected bypass detection")
	}
}

func TestOutputValidationBlocksForbiddenClaims(t *testing.T) {
	violations := compliance.ValidateOutputText("This product is certified safe", nil)
	if len(violations) == 0 {
		t.Fatal("expected output violation")
	}
}

func TestPIIRedaction(t *testing.T) {
	redacted := compliance.RedactPII("contact me at user@example.com")
	if redacted == "contact me at user@example.com" {
		t.Fatal("expected email redaction")
	}
}

func TestDataClassification(t *testing.T) {
	class := compliance.ClassifyField("email", "user@example.com")
	if class != compliance.ClassPII {
		t.Fatalf("expected PII class, got %s", class)
	}
	_ = domain.ProfileKVKK
}
