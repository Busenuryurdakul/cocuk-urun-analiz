package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/fetch"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/reviewinsight"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ToolInput struct {
	OrganizationID primitive.ObjectID
	RunID          primitive.ObjectID
	ProductID      primitive.ObjectID
	ActorID        primitive.ObjectID
	Role           domain.OrgRole
	Resolved       ResolvedInput
	SourceURL      string
	ClaimID        string
	ClaimText      string
	EvidenceIDs    []string
	RecallIDs      []string
	ReviewIDs      []string
}

type ToolOutput struct {
	Status  string
	Payload map[string]any
}

type Executor struct {
	Registry   *Registry
	Compliance *compliance.Engine
	Evidence   *evidence.Service
	Safety     *safety.Service
	Reviews    *reviewinsight.Service
}

func (e *Executor) Execute(ctx context.Context, toolName string, input ToolInput) (*ToolOutput, error) {
	def, ok := e.Registry.Get(toolName)
	if !ok || def.Availability != ToolAvailable {
		return nil, ErrToolUnavailable
	}
	if !roleAllowsTool(input.Role, def.MinRole) {
		return nil, ErrAuthorizationDenied
	}

	switch toolName {
	case "policy_evaluator":
		return e.runPolicyEvaluator(ctx, input)
	case "compliance_checker":
		return e.runComplianceChecker(ctx, input)
	case "fetch_policy_checker":
		return e.runFetchPolicyChecker(input)
	case "review_sampler":
		return e.runReviewSampler(input)
	case "dataset_validator":
		return e.runDatasetValidator(input)
	case "pii_redactor":
		return e.runPIIRedactor(input)
	case "evidence_validator":
		return e.runEvidenceValidator(ctx, input)
	case "safety_analyzer":
		return e.runSafetyAnalyzer(ctx, input)
	case "review_analyzer":
		return e.runReviewAnalyzer(ctx, input)
	default:
		return nil, ErrToolUnavailable
	}
}

func roleAllowsTool(role domain.OrgRole, minRole string) bool {
	switch minRole {
	case "ANALYST":
		return rbac.CanStartAnalysisRun(role)
	case "VIEWER":
		return rbac.CanReadAnalysisRun(role)
	default:
		return rbac.CanStartAnalysisRun(role)
	}
}

func (e *Executor) runPolicyEvaluator(ctx context.Context, input ToolInput) (*ToolOutput, error) {
	orgID := input.OrganizationID
	result, err := e.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "start_analysis_run",
		OrganizationID: orgID,
		UserID:         input.ActorID,
		Purpose:        domain.ConsentPurposeDataProcessing,
	})
	if err != nil {
		return nil, err
	}
	return &ToolOutput{
		Status: "OK",
		Payload: map[string]any{
			"allowed":       result.Allowed,
			"policyVersion": result.PolicyVersion,
			"profile":       string(result.Profile),
		},
	}, nil
}

func (e *Executor) runComplianceChecker(ctx context.Context, input ToolInput) (*ToolOutput, error) {
	fields := map[string]string{
		"productId": input.ProductID.Hex(),
	}
	result, err := e.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "analysis_tool_loop",
		OrganizationID: input.OrganizationID,
		UserID:         input.ActorID,
		Purpose:        domain.ConsentPurposeDataProcessing,
		Fields:         fields,
	})
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, ErrComplianceRejected
	}
	return &ToolOutput{
		Status: "OK",
		Payload: map[string]any{
			"allowed":       true,
			"policyVersion": result.PolicyVersion,
			"violations":    result.Violations,
		},
	}, nil
}

func (e *Executor) runFetchPolicyChecker(input ToolInput) (*ToolOutput, error) {
	url := input.SourceURL
	if url == "" {
		return &ToolOutput{
			Status: "SKIPPED",
			Payload: map[string]any{
				"reason": "no_source_url_in_analysis_context",
			},
		}, nil
	}
	if _, err := fetch.ValidateURLStructure(url, nil); err != nil {
		return nil, err
	}
	return &ToolOutput{Status: "OK", Payload: map[string]any{"urlValid": true}}, nil
}

func (e *Executor) runReviewSampler(input ToolInput) (*ToolOutput, error) {
	reviews := input.Resolved.MarketplaceReviews
	total := len(reviews)
	sampled := total
	if sampled > MaxReviewSample {
		sampled = MaxReviewSample
	}
	sample := reviews[:sampled]
	ids := make([]string, 0, sampled)
	for _, rv := range sample {
		ids = append(ids, rv.ID.Hex())
	}
	return &ToolOutput{
		Status: "OK",
		Payload: map[string]any{
			"totalAvailable": total,
			"sampled":        sampled,
			"reviewIds":      ids,
			"sourceType":     domain.SourceTypeMarketplaceReview,
		},
	}, nil
}

func (e *Executor) runDatasetValidator(input ToolInput) (*ToolOutput, error) {
	summary := map[string]int{}
	for _, rv := range input.Resolved.MarketplaceReviews {
		el := dataset.EvaluateEligibility(dataset.EligibilityInput{
			ModerationStatus:  rv.ModerationStatus,
			QualityStatus:     rv.QualityStatus,
			PIIStatus:         rv.PIIStatus,
			ProvenanceStatus:  rv.ProvenanceStatus,
			LicenseStatus:     rv.LicenseStatus,
			UsageRightsStatus: rv.UsageRightsStatus,
			SpamStatus:        rv.SpamStatus,
			DuplicateStatus:   rv.DuplicateStatus,
		})
		summary[string(el)]++
	}
	for _, ux := range input.Resolved.UserExperiences {
		el := dataset.EvaluateUGCEligibility(ux.ModerationStatus, ux.QualityStatus, ux.PIIStatus)
		summary[string(el)]++
	}
	return &ToolOutput{
		Status: "OK",
		Payload: map[string]any{
			"marketplaceReviewCount": input.Resolved.MarketplaceReviewCount,
			"ugcCount":               input.Resolved.UGCCount,
			"eligibilitySummary":     summary,
		},
	}, nil
}

func (e *Executor) runSafetyAnalyzer(ctx context.Context, input ToolInput) (*ToolOutput, error) {
	if e.Safety == nil {
		return nil, ErrToolUnavailable
	}
	if err := ValidateSafetyAnalyzerInput(input); err != nil {
		return nil, err
	}
	result, err := e.Safety.Analyze(ctx, safety.AnalyzeInput{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.RunID,
		ProductID:      input.ProductID,
		EvidenceIDs:    input.EvidenceIDs,
		RecallIDs:      input.RecallIDs,
	})
	if err != nil {
		return nil, err
	}
	findings := make([]map[string]any, 0, len(result.Findings))
	for _, f := range result.Findings {
		ids := f.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		findings = append(findings, map[string]any{
			"type":        f.Type,
			"severity":    f.Severity,
			"confidence":  f.Confidence,
			"evidenceIds": ids,
			"rationale":   f.Rationale,
		})
	}
	issues := result.Issues
	if issues == nil {
		issues = []string{}
	}
	payload := map[string]any{"findings": findings, "issues": issues}
	if err := ValidateToolOutput("safety_analyzer", "OK", payload); err != nil {
		return nil, err
	}
	return &ToolOutput{Status: "OK", Payload: payload}, nil
}

func (e *Executor) runReviewAnalyzer(ctx context.Context, input ToolInput) (*ToolOutput, error) {
	if e.Reviews == nil {
		return nil, ErrToolUnavailable
	}
	if err := ValidateReviewAnalyzerInput(input); err != nil {
		return nil, err
	}
	result, err := e.Reviews.Analyze(ctx, reviewinsight.AnalyzeInput{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.RunID,
		ProductID:      input.ProductID,
		ReviewIDs:      input.ReviewIDs,
		Reviews:        input.Resolved.MarketplaceReviews,
		Experiences:    input.Resolved.UserExperiences,
	})
	if err != nil {
		return nil, err
	}
	issues := result.Issues
	if issues == nil {
		issues = []string{}
	}
	payload := map[string]any{
		"positiveSignals":    result.PositiveSignals,
		"negativeSignals":    result.NegativeSignals,
		"safetySignals":      result.SafetySignals,
		"qualityIssues":      result.QualityIssues,
		"durabilityIssues":   result.DurabilityIssues,
		"usabilityIssues":    result.UsabilityIssues,
		"assemblyIssues":     result.AssemblyIssues,
		"ageMismatchSignals": result.AgeMismatchSignals,
		"repeatedComplaints": result.RepeatedComplaints,
		"issues":             issues,
	}
	if err := ValidateToolOutput("review_analyzer", "OK", payload); err != nil {
		return nil, err
	}
	return &ToolOutput{Status: "OK", Payload: payload}, nil
}

func (e *Executor) runEvidenceValidator(ctx context.Context, input ToolInput) (*ToolOutput, error) {
	if e.Evidence == nil {
		return nil, ErrToolUnavailable
	}
	if err := ValidateEvidenceValidatorInput(input); err != nil {
		return nil, err
	}
	result, err := e.Evidence.ValidateClaim(ctx, evidence.ValidateClaimInput{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.RunID,
		ClaimID:        input.ClaimID,
		ClaimText:      input.ClaimText,
		EvidenceIDs:    input.EvidenceIDs,
	})
	if err != nil {
		return nil, err
	}
	issues := result.Issues
	if issues == nil {
		issues = []string{}
	}
	ids := result.EvidenceIDs
	if ids == nil {
		ids = []string{}
	}
	payload := map[string]any{
		"claimId":     result.ClaimID,
		"status":      string(result.Status),
		"evidenceIds": ids,
		"issues":      issues,
	}
	if err := ValidateToolOutput("evidence_validator", "OK", payload); err != nil {
		return nil, err
	}
	return &ToolOutput{Status: "OK", Payload: payload}, nil
}

func (e *Executor) runPIIRedactor(input ToolInput) (*ToolOutput, error) {
	meta := map[string]string{
		"productName": fmt.Sprint(input.Resolved.Product.Name.Value),
	}
	redacted := compliance.RedactFields(meta)
	return &ToolOutput{
		Status: "OK",
		Payload: map[string]any{
			"redactedMetadata": redacted,
		},
	}, nil
}

func HashPayload(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type AuthorizationGrant struct {
	Nonce           string
	OrganizationID  primitive.ObjectID
	UserID          primitive.ObjectID
	AnalysisRunID   primitive.ObjectID
	ToolExecutionID primitive.ObjectID
	ToolName        string
	ToolVersion     string
	InputHash       string
	ExpiresAt       time.Time
}

type Authorizer struct {
	Registry *Registry
	Security *repository.SecurityEventRepository
}

func (a *Authorizer) Authorize(ctx context.Context, role domain.OrgRole, input ToolInput, toolName, toolVersion, inputHash string) (*AuthorizationGrant, error) {
	def, ok := a.Registry.Get(toolName)
	if !ok || def.Availability != ToolAvailable {
		uid := input.ActorID
		oid := input.OrganizationID
		_ = a.Security.Record(ctx, domain.SecurityEvent{
			OrganizationID: &oid,
			UserID:         &uid,
			EventType:      domain.EventUnauthorizedTool,
			Severity:       domain.SeverityWarning,
			Details:        map[string]string{"tool": toolName, "reason": "unavailable"},
		})
		return nil, ErrToolUnavailable
	}
	if !roleAllowsTool(role, def.MinRole) {
		uid := input.ActorID
		oid := input.OrganizationID
		_ = a.Security.Record(ctx, domain.SecurityEvent{
			OrganizationID: &oid,
			UserID:         &uid,
			EventType:      domain.EventUnauthorizedTool,
			Severity:       domain.SeverityWarning,
			Details:        map[string]string{"tool": toolName, "reason": "rbac"},
		})
		return nil, ErrAuthorizationDenied
	}

	execID := primitive.NewObjectID()
	nonce := HashPayload(map[string]any{
		"run": input.RunID.Hex(), "tool": toolName, "input": inputHash, "ts": time.Now().UTC().UnixNano(),
	})
	return &AuthorizationGrant{
		Nonce:           nonce,
		OrganizationID:  input.OrganizationID,
		UserID:          input.ActorID,
		AnalysisRunID:   input.RunID,
		ToolExecutionID: execID,
		ToolName:        toolName,
		ToolVersion:     toolVersion,
		InputHash:       inputHash,
		ExpiresAt:       time.Now().UTC().Add(5 * time.Minute),
	}, nil
}

func (a *Authorizer) ValidateGrant(_ *AuthorizationGrant, _ string) error {
	return nil
}
