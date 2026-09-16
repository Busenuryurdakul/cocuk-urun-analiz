package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEnforcerPreCallRedactsPII(t *testing.T) {
	enforcer := &Enforcer{}
	result, err := enforcer.PreCall(context.Background(), &domain.Organization{
		ComplianceProfile:       string(domain.ProfileKVKK),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}, EnforcementInput{
		UserPrompt: "Contact user@example.com for details.",
	})
	if err != nil {
		t.Fatalf("precall: %v", err)
	}
	if !result.RedactionApplied {
		t.Fatal("expected PII redaction")
	}
	if strings.Contains(result.UserPrompt, "user@example.com") {
		t.Fatalf("expected redacted prompt, got %q", result.UserPrompt)
	}
	if result.Compliance.Profile != domain.ProfileKVKK {
		t.Fatalf("expected KVKK profile, got %q", result.Compliance.Profile)
	}
	if result.Compliance.ReflexInstruction == "" {
		t.Fatal("expected reflex instruction")
	}
}

func TestEnforcerPostCallBlocksForbiddenClaims(t *testing.T) {
	enforcer := &Enforcer{}
	org := &domain.Organization{
		ID:                      primitive.NewObjectID(),
		ComplianceProfile:       string(domain.ProfileGDPR),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}
	complianceCtx := ComplianceContext{
		Profile:           domain.ProfileGDPR,
		PolicyVersion:     compliance.PlatformDefaultPolicyVersion,
		ReflexVersion:     compliance.ComplianceReflexVersion(),
		ForbiddenPatterns: []string{"certified safe"},
	}
	_, _, err := enforcer.PostCall(context.Background(), org, primitive.NilObjectID,
		"This product is certified safe for all children.", complianceCtx, "analysis")
	if err == nil {
		t.Fatal("expected output validation block")
	}
}

func TestEnforcerPostCallSanitizesReviewOutput(t *testing.T) {
	enforcer := &Enforcer{}
	org := &domain.Organization{
		ID:                      primitive.NewObjectID(),
		ComplianceProfile:       string(domain.ProfileGDPR),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}
	complianceCtx := ComplianceContext{
		Profile:           domain.ProfileGDPR,
		ForbiddenPatterns: []string{"certified safe"},
	}
	result, sanitized, err := enforcer.PostCall(context.Background(), org, primitive.NilObjectID,
		"Worker incorrectly claimed certified safe; evidence does not support that.", complianceCtx, "review")
	if err != nil {
		t.Fatalf("review postcall: %v", err)
	}
	if result.SafetyResult != "output_sanitized" {
		t.Fatalf("expected sanitized result, got %+v", result)
	}
	if strings.Contains(sanitized, "certified safe") {
		t.Fatalf("expected forbidden phrase removed, got %q", sanitized)
	}
}

func TestEnforcerPostCallAllowsCleanOutput(t *testing.T) {
	enforcer := &Enforcer{}
	org := &domain.Organization{
		ID:                      primitive.NewObjectID(),
		ComplianceProfile:       string(domain.ProfileBoth),
		CompliancePolicyVersion: compliance.PlatformDefaultPolicyVersion,
	}
	complianceCtx := ComplianceContext{
		Profile:           domain.ProfileBoth,
		ForbiddenPatterns: []string{"certified safe"},
	}
	result, _, err := enforcer.PostCall(context.Background(), org, primitive.NilObjectID,
		"Evidence suggests moderate risk; further verification recommended.", complianceCtx, "analysis")
	if err != nil {
		t.Fatalf("postcall: %v", err)
	}
	if !result.Allowed || result.SafetyResult != "PASS" {
		t.Fatalf("expected pass, got %+v", result)
	}
}

func TestBuildSystemInstructionCombinesPersonaAndReflex(t *testing.T) {
	got := BuildSystemInstruction("Base persona.", "Reflex block.")
	if !strings.Contains(got, "Base persona.") || !strings.Contains(got, "Reflex block.") {
		t.Fatalf("unexpected combined instruction: %q", got)
	}
}
