package compliance

import (
	"context"
	"fmt"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EvaluateRequest is the deterministic compliance input for Phase 3 API operations.
type EvaluateRequest struct {
	Operation      string
	OrganizationID primitive.ObjectID
	UserID         primitive.ObjectID
	Purpose        domain.ConsentPurpose
	ProfileInput   string
	Fields         map[string]string
	OutputText     string
}

type EvaluateResult struct {
	Allowed          bool
	Violations       []string
	ClassifiedFields map[string]DataClass
	RedactedFields   map[string]string
	PolicyVersion    string
	Profile          domain.ComplianceProfile
}

type Engine struct {
	PolicyRepo *PolicyRepository
	Consent    *ConsentService
	Events     *repository.ComplianceEventRepository
	Security   *repository.SecurityEventRepository
	Orgs       *repository.OrganizationRepository
}

func (e *Engine) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResult, error) {
	result := &EvaluateResult{Allowed: true, ClassifiedFields: map[string]DataClass{}, RedactedFields: map[string]string{}}

	if DetectBypassAttempt(req.ProfileInput) || DetectBypassAttempt(valuesFromMap(req.Fields)...) {
		e.recordBypass(ctx, req, "bypass_pattern")
		return nil, ErrBypassAttempt
	}

	org, err := e.Orgs.FindByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	profile := domain.ComplianceProfile(org.ComplianceProfile)
	if req.ProfileInput != "" {
		p, err := ValidateProfile(req.ProfileInput)
		if err != nil {
			e.recordBypass(ctx, req, "invalid_profile_input")
			return nil, err
		}
		profile = p
	}
	if !ProfileAlwaysOn(profile) {
		e.recordBypass(ctx, req, "profile_not_always_on")
		return nil, ErrInvalidProfile
	}
	result.Profile = profile

	policy, err := e.PolicyRepo.ResolveActive(ctx, org)
	if err != nil {
		return nil, err
	}
	result.PolicyVersion = policy.Version

	if err := ValidateMinimization(req.Operation, req.Fields); err != nil {
		result.Allowed = false
		result.Violations = append(result.Violations, err.Error())
		e.recordEval(ctx, req, policy.Version, "FAIL", "DATA_MINIMIZATION")
		return result, ErrComplianceViolation
	}

	result.ClassifiedFields = ClassifyFields(req.Fields)
	result.RedactedFields = RedactFields(req.Fields)

	if err := e.checkRequiredConsents(ctx, req, policy, result); err != nil {
		return result, err
	}

	if req.OutputText != "" {
		violations := ValidateOutputText(req.OutputText, policy.Rules.ForbiddenClaimPatterns)
		if len(violations) > 0 {
			result.Allowed = false
			result.Violations = violations
			e.recordEval(ctx, req, policy.Version, "FAIL", "OUTPUT_VALIDATION")
			return result, ErrComplianceViolation
		}
	}

	e.recordEval(ctx, req, policy.Version, "PASS", req.Operation)
	return result, nil
}

var protectedConsentOperations = map[string]bool{
	"invite_member":             true,
	"update_compliance_profile": true,
	"create_organization":       true,
}

func (e *Engine) checkRequiredConsents(ctx context.Context, req EvaluateRequest, policy *domain.CompliancePolicyVersion, result *EvaluateResult) error {
	if req.Purpose != "" {
		return e.requireConsent(ctx, req, policy.Version, req.Purpose, result)
	}
	if !protectedConsentOperations[req.Operation] {
		return nil
	}
	for _, purposeRaw := range policy.Rules.RequireConsentPurposes {
		purpose := domain.ConsentPurpose(purposeRaw)
		if err := e.requireConsent(ctx, req, policy.Version, purpose, result); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) requireConsent(ctx context.Context, req EvaluateRequest, policyVersion string, purpose domain.ConsentPurpose, result *EvaluateResult) error {
	var orgPtr *primitive.ObjectID
	if req.OrganizationID != primitive.NilObjectID {
		oid := req.OrganizationID
		orgPtr = &oid
	}
	if err := e.Consent.RequireConsent(ctx, req.UserID, orgPtr, purpose); err != nil {
		result.Allowed = false
		result.Violations = append(result.Violations, fmt.Sprintf("consent:%s", purpose))
		e.recordEval(ctx, req, policyVersion, "FAIL", "CONSENT")
		return ErrConsentRequired
	}
	return nil
}

func (e *Engine) recordBypass(ctx context.Context, req EvaluateRequest, reason string) {
	uid := req.UserID
	oid := req.OrganizationID
	_ = e.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventComplianceBypassAttempt,
		Severity:       domain.SeverityCritical,
		Details:        map[string]string{"reason": reason, "operation": req.Operation},
	})
	_ = e.Events.Record(ctx, domain.ComplianceEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      "BYPASS_ATTEMPT",
		RuleID:         reason,
		Result:         "FAIL",
		Details:        map[string]string{"operation": req.Operation},
	})
}

func (e *Engine) recordEval(ctx context.Context, req EvaluateRequest, version, result, ruleID string) {
	uid := req.UserID
	oid := req.OrganizationID
	_ = e.Events.Record(ctx, domain.ComplianceEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      "POLICY_EVALUATION",
		RuleID:         ruleID,
		Result:         result,
		Details: map[string]string{
			"operation": req.Operation,
			"version":   version,
		},
	})
}

func valuesFromMap(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// AgentHook is an interface-only contract for later agent phases.
type AgentHook interface {
	PreCheck(ctx context.Context, req EvaluateRequest) (*EvaluateResult, error)
	PostCheck(ctx context.Context, req EvaluateRequest) (*EvaluateResult, error)
}
