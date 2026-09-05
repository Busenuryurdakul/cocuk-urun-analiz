package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EnforcementInput struct {
	OrganizationID     primitive.ObjectID
	UserID             primitive.ObjectID
	ConfigSnapshotID   *primitive.ObjectID
	UserPrompt         string
	ExternalContent    []string
	PersonaInstruction string
}

type ComplianceContext struct {
	Profile             domain.ComplianceProfile
	PolicyVersion       string
	ReflexVersion       string
	ReflexInstruction   string
	ForbiddenPatterns   []string
	Policy              *domain.CompliancePolicyVersion
}

type EnforcementResult struct {
	UserPrompt          string
	PersonaInstruction  string
	RedactionApplied    bool
	Blocked             bool
	BlockReason         string
	Compliance          ComplianceContext
}

type PostCallResult struct {
	Allowed      bool
	Violations   []string
	SafetyResult string
}

// Enforcer applies platform security and compliance before and after provider calls.
type Enforcer struct {
	Compliance *compliance.Engine
	PolicyRepo *compliance.PolicyRepository
	Snapshots  *repository.ConfigSnapshotRepository
}

func (e *Enforcer) ResolveComplianceContext(ctx context.Context, org *domain.Organization, snapshotID *primitive.ObjectID) (ComplianceContext, error) {
	out := ComplianceContext{
		ReflexVersion: compliance.ComplianceReflexVersion(),
	}
	if org == nil {
		return out, nil
	}

	profile := domain.ComplianceProfile(org.ComplianceProfile)
	if !compliance.ProfileAlwaysOn(profile) {
		profile = domain.ProfileBoth
	}
	out.Profile = profile

	versionPin := strings.TrimSpace(org.CompliancePolicyVersion)
	if snapshotID != nil && e.Snapshots != nil {
		snap, err := e.Snapshots.FindByID(ctx, *snapshotID)
		if err == nil && strings.TrimSpace(snap.CompliancePolicyVersion) != "" {
			versionPin = strings.TrimSpace(snap.CompliancePolicyVersion)
		}
	}

	resolveOrg := *org
	resolveOrg.CompliancePolicyVersion = versionPin

	if e.PolicyRepo != nil {
		policy, err := e.PolicyRepo.ResolveActive(ctx, &resolveOrg)
		if err == nil {
			out.Policy = policy
			out.PolicyVersion = policy.Version
			out.ForbiddenPatterns = policy.Rules.ForbiddenClaimPatterns
		} else if versionPin != "" {
			out.PolicyVersion = versionPin
		}
	} else if versionPin != "" {
		out.PolicyVersion = versionPin
	}

	out.ReflexInstruction = compliance.ComplianceReflexInstruction(profile, out.ReflexVersion)
	return out, nil
}

func (e *Enforcer) PreCall(ctx context.Context, org *domain.Organization, in EnforcementInput) (EnforcementResult, error) {
	complianceCtx, err := e.ResolveComplianceContext(ctx, org, in.ConfigSnapshotID)
	if err != nil {
		return EnforcementResult{}, err
	}

	out := EnforcementResult{
		UserPrompt:         in.UserPrompt,
		PersonaInstruction: in.PersonaInstruction,
		Compliance:         complianceCtx,
	}
	if compliance.DetectBypassAttempt(in.UserPrompt, in.PersonaInstruction) {
		out.Blocked = true
		out.BlockReason = "compliance_bypass_attempt"
		return out, ErrPromptInjection
	}
	for _, ext := range in.ExternalContent {
		if detectExternalInjection(ext) {
			out.Blocked = true
			out.BlockReason = "external_content_injection"
			return out, ErrPromptInjection
		}
	}
	if detectExternalInjection(in.UserPrompt) {
		out.Blocked = true
		out.BlockReason = "user_prompt_injection"
		return out, ErrPromptInjection
	}
	if org != nil {
		redacted := compliance.RedactPII(in.UserPrompt)
		if redacted != in.UserPrompt {
			out.UserPrompt = redacted
			out.RedactionApplied = true
		}
	}
	return out, nil
}

func (e *Enforcer) PostCall(ctx context.Context, org *domain.Organization, userID primitive.ObjectID, outputText string, complianceCtx ComplianceContext) (PostCallResult, error) {
	result := PostCallResult{Allowed: true, SafetyResult: "PASS"}
	if strings.TrimSpace(outputText) == "" {
		return result, nil
	}

	patterns := complianceCtx.ForbiddenPatterns
	if complianceCtx.Policy != nil {
		patterns = complianceCtx.Policy.Rules.ForbiddenClaimPatterns
	}
	violations := compliance.ValidateOutputText(outputText, patterns)
	if len(violations) > 0 {
		result.Allowed = false
		result.Violations = violations
		result.SafetyResult = "output_validation_fail"
		if e.Compliance != nil && org != nil {
			_, _ = e.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
				Operation:      "llm_complete_output",
				OrganizationID: org.ID,
				UserID:         userID,
				OutputText:     outputText,
			})
		}
		return result, ErrComplianceBlocked
	}

	if e.Compliance != nil && org != nil {
		_, _ = e.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
			Operation:      "llm_complete_output",
			OrganizationID: org.ID,
			UserID:         userID,
			OutputText:     outputText,
		})
	}
	return result, nil
}

func BuildSystemInstruction(personaInstruction, reflexInstruction string) string {
	personaInstruction = strings.TrimSpace(personaInstruction)
	reflexInstruction = strings.TrimSpace(reflexInstruction)
	if reflexInstruction == "" {
		return personaInstruction
	}
	if personaInstruction == "" {
		return reflexInstruction
	}
	return personaInstruction + "\n\n" + reflexInstruction
}

func detectExternalInjection(text string) bool {
	lower := strings.ToLower(text)
	patterns := []string{
		"ignore previous instructions",
		"ignore all previous",
		"system prompt override",
		"you are now",
		"disregard compliance",
		"override platform security",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func HashContent(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
