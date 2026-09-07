package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/reviewinsight"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid final analysis input")
)

type Service struct {
	Runs        *repository.AnalysisRunRepository
	Evidences   *repository.EvidenceRepository
	Validations *repository.EvidenceClaimValidationRepository
	Findings    *repository.SafetyFindingRepository
	Matches     *repository.RecallMatchRepository
	Reviews     *reviewinsight.Service
	Products    *repository.ProductRepository
	Mappings    *repository.ProductSourceMappingRepository
	Tenant      *tenant.Guard
}

type FinalizeInput struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	Worker         domain.AnalysisLLMResult
	Reviewer       domain.AnalysisLLMResult
	ComplianceMax  string
}

func (s *Service) Finalize(ctx context.Context, input FinalizeInput) (*domain.FinalAnalysisResult, error) {
	if input.OrganizationID.IsZero() || input.AnalysisRunID.IsZero() {
		return nil, ErrInvalidInput
	}
	run, err := s.Runs.FindByID(ctx, input.OrganizationID, input.AnalysisRunID)
	if err != nil {
		return nil, err
	}
	evidence, err := s.Evidences.ListByAnalysisRun(ctx, input.OrganizationID, input.AnalysisRunID, 100)
	if err != nil {
		return nil, err
	}
	validations, err := s.Validations.ListByAnalysisRun(ctx, input.OrganizationID, input.AnalysisRunID, 50)
	if err != nil {
		return nil, err
	}
	findings, err := s.Findings.ListByAnalysisRun(ctx, input.OrganizationID, input.AnalysisRunID, 50)
	if err != nil {
		return nil, err
	}
	matches, err := s.Matches.ListByAnalysisRun(ctx, input.OrganizationID, input.AnalysisRunID, 50)
	if err != nil {
		return nil, err
	}
	reviewCount := 0
	var insight *domain.ReviewInsightRecord
	if s.Reviews != nil {
		insight, _ = s.Reviews.LatestForRun(ctx, input.OrganizationID, input.AnalysisRunID)
		if insight != nil {
			reviewCount = insight.ReviewCount
		}
	}
	hasIdentity := false
	if s.Products != nil {
		if product, perr := s.Products.FindByID(ctx, input.OrganizationID, run.ProductID); perr == nil && product != nil {
			hasIdentity = stringify(product.Name.Value) != "" || stringify(product.Brand.Value) != ""
		}
	}

	guard := ApplyHallucinationGuard(GuardInput{
		WorkerText: input.Worker.Output, ReviewerText: input.Reviewer.Output,
		Evidence: evidence, Validations: validations, Findings: findings, Matches: matches, ReviewCoverage: reviewCount,
	})
	conf := ComputeConfidence(ConfidenceInput{
		Evidence: evidence, Validations: validations, Findings: findings, Matches: guard.Matches,
		ReviewCount: reviewCount, HasIdentity: hasIdentity, Hallucination: guard.Flags,
		ReviewerPassed: !hasFlag(guard.Flags, string(domain.FlagUnsupportedClaim), string(domain.FlagInventedRecall)),
	})
	dec := Decide(DecisionInput{
		Findings: findings, Matches: guard.Matches, Evidence: evidence, Validations: validations,
		Flags: guard.Flags, ComplianceMax: input.ComplianceMax,
	})

	now := time.Now().UTC()
	if strings.TrimSpace(input.Worker.Output) == "" {
		input.Worker.Output = "Worker output was empty; deterministic analysis used persisted tool results only."
	}
	if strings.TrimSpace(input.Reviewer.Output) == "" {
		input.Reviewer.Output = "Reviewer output was empty; deterministic analysis used persisted tool results only."
	}
	input.Worker.Role = "worker"
	if input.Worker.CreatedAt.IsZero() {
		input.Worker.CreatedAt = now
	}
	input.Reviewer.Role = "reviewer"
	if input.Reviewer.CreatedAt.IsZero() {
		input.Reviewer.CreatedAt = now
	}
	if input.Reviewer.Flags == nil {
		input.Reviewer.Flags = guard.Flags
	}

	final := &domain.FinalAnalysisResult{
		SchemaVersion:      domain.FinalAnalysisSchemaVersion,
		Summary:            buildSummary(dec, guard, insight),
		OverallRisk:        dec.OverallRisk,
		Confidence:         conf.Score,
		ConfidenceFactors:  conf.Factors,
		PositiveSignals:    signalsOrEmpty(insight, true),
		NegativeSignals:    signalsOrEmpty(insight, false),
		ReviewInsights:     flattenInsights(insight),
		SafetyFindings:     findings,
		Recalls:            guard.Matches,
		Decision:           dec.Decision,
		Recommendation:     dec.Recommendation,
		Limitations:        guard.Limitations,
		HallucinationFlags: guard.Flags,
		Worker:             input.Worker,
		Reviewer:           input.Reviewer,
		CreatedAt:          now,
	}
	run.WorkerResult = &input.Worker
	run.ReviewerResult = &input.Reviewer
	run.FinalResult = final
	run.FinalResultSchemaVersion = domain.FinalAnalysisSchemaVersion
	if err := s.Runs.Update(ctx, input.OrganizationID, run); err != nil {
		return nil, err
	}
	return final, nil
}

func (s *Service) GetFinalForActor(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID) (*domain.FinalAnalysisResult, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	run, err := s.Runs.FindByID(ctx, organizationID, analysisRunID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return run.FinalResult, nil
}

func (s *Service) ListReviewInsightsForActor(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID) ([]domain.ReviewSignal, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	if _, err := s.Runs.FindByID(ctx, organizationID, analysisRunID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []domain.ReviewSignal{}, nil
		}
		return nil, err
	}
	if s.Reviews == nil {
		return []domain.ReviewSignal{}, nil
	}
	insight, err := s.Reviews.LatestForRun(ctx, organizationID, analysisRunID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []domain.ReviewSignal{}, nil
		}
		return nil, err
	}
	return flattenInsights(insight), nil
}

func buildSummary(dec DecisionResult, guard GuardResult, insight *domain.ReviewInsightRecord) string {
	parts := []string{dec.Recommendation}
	if insight != nil && insight.ReviewCount == 0 {
		parts = append(parts, "No approved reviews were available.")
	}
	if hasFlag(guard.Flags, string(domain.FlagInventedRecall)) {
		parts = append(parts, "Unverified recall language was removed from the confirmed recall list.")
	}
	return strings.Join(parts, " ")
}

func signalsOrEmpty(insight *domain.ReviewInsightRecord, positive bool) []domain.ReviewSignal {
	if insight == nil {
		return []domain.ReviewSignal{}
	}
	if positive {
		return insight.PositiveSignals
	}
	return insight.NegativeSignals
}

func flattenInsights(insight *domain.ReviewInsightRecord) []domain.ReviewSignal {
	if insight == nil {
		return []domain.ReviewSignal{}
	}
	out := make([]domain.ReviewSignal, 0)
	out = append(out, insight.PositiveSignals...)
	out = append(out, insight.NegativeSignals...)
	out = append(out, insight.SafetySignals...)
	out = append(out, insight.RepeatedComplaints...)
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
