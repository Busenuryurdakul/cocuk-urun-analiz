package safety

import (
	"context"
	"errors"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid safety input")
)

type SourceAdapter interface {
	Source() string
	SearchRecalls(ctx context.Context, identity recall.Identity) ([]recall.Record, error)
}

type Service struct {
	Findings  *repository.SafetyFindingRepository
	Matches   *repository.RecallMatchRepository
	Runs      *repository.AnalysisRunRepository
	Products  *repository.ProductRepository
	Mappings  *repository.ProductSourceMappingRepository
	Evidences *evidence.Service
	Tenant    *tenant.Guard
	Adapters  []SourceAdapter
}

type AnalyzeInput struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	ProductID      primitive.ObjectID
	EvidenceIDs    []string
	RecallIDs      []string
}

type AnalyzeResult struct {
	Findings []FindingOutput
	Issues   []string
}

type FindingOutput struct {
	Type        string
	Severity    string
	Confidence  float64
	EvidenceIDs []string
	Rationale   string
}

func (s *Service) Analyze(ctx context.Context, input AnalyzeInput) (*AnalyzeResult, error) {
	if input.OrganizationID.IsZero() || input.AnalysisRunID.IsZero() || input.ProductID.IsZero() {
		return nil, ErrInvalidInput
	}
	run, err := s.Runs.FindByID(ctx, input.OrganizationID, input.AnalysisRunID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, evidence.ErrForeignAnalysisRun
		}
		return nil, err
	}
	product, err := s.Products.FindByID(ctx, input.OrganizationID, input.ProductID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, evidence.ErrForeignProduct
		}
		return nil, err
	}
	if run.ProductID != product.ID {
		return nil, evidence.ErrProductRunMismatch
	}

	issues := make([]string, 0)
	loaded, loadIssues, err := s.loadScopedEvidence(ctx, input)
	if err != nil {
		return nil, err
	}
	issues = append(issues, loadIssues...)

	mappings, err := s.Mappings.FindByProductID(ctx, input.OrganizationID, input.ProductID)
	if err != nil {
		return nil, err
	}
	identity := IdentityFromProduct(product, mappings)

	candidates, searchIssues := s.collectRecalls(ctx, identity, input.RecallIDs)
	issues = append(issues, searchIssues...)

	result := &AnalyzeResult{Findings: []FindingOutput{}, Issues: issues}
	if len(loaded) == 0 && len(candidates) == 0 {
		result.Issues = appendUnique(result.Issues, "NO_EVIDENCE")
		if err := s.persistInsufficient(ctx, input); err != nil {
			return nil, err
		}
		return result, nil
	}

	for _, rec := range candidates {
		match := recall.Match(identity, rec)
		if err := s.persistMatch(ctx, input, rec, match); err != nil {
			return nil, err
		}
		if !recall.Confirmed(match) {
			if match.Method == recall.MethodFuzzyName || match.RequiresReview {
				result.Issues = appendUnique(result.Issues, "WEAK_RECALL_MATCH")
			}
			continue
		}
		ev, err := s.createRecallEvidence(ctx, input, rec, match)
		if err != nil {
			return nil, err
		}
		finding, err := s.persistFinding(ctx, input, domain.SafetyRecall, domain.SafetySeverityCritical, match.Confidence, []primitive.ObjectID{ev.ID}, rec.Source, "Confirmed official recall matched by "+match.Method+".")
		if err != nil {
			return nil, err
		}
		result.Findings = append(result.Findings, toFindingOutput(finding))
	}

	for _, ev := range loaded {
		if ev.SourceType == "OFFICIAL_RECALL" || ev.SourceType == "OFFICIAL_SAFETY_NOTICE" {
			if ev.Reliability >= recall.ConfirmedMatchMinConfidence {
				finding, err := s.persistFinding(ctx, input, domain.SafetyRecall, domain.SafetySeverityCritical, ev.Reliability, []primitive.ObjectID{ev.ID}, ev.Source, "Official recall evidence already persisted.")
				if err != nil {
					return nil, err
				}
				result.Findings = append(result.Findings, toFindingOutput(finding))
			}
			continue
		}
		if ev.Reliability < 0.8 {
			result.Issues = appendUnique(result.Issues, "UNSUPPORTED_EVIDENCE")
			continue
		}
		ftype := classifyEvidence(ev)
		if ftype == "" {
			continue
		}
		finding, err := s.persistFinding(ctx, input, ftype, domain.SafetySeverityHigh, ev.Reliability, []primitive.ObjectID{ev.ID}, ev.Source, "High-severity supported evidence linked to persisted source.")
		if err != nil {
			return nil, err
		}
		result.Findings = append(result.Findings, toFindingOutput(finding))
	}

	if len(result.Findings) == 0 && len(loaded) == 0 {
		result.Issues = appendUnique(result.Issues, "NO_EVIDENCE")
	}
	return result, nil
}

func (s *Service) ListFindingsForActor(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.SafetyFinding, error) {
	if err := s.authorizeRead(ctx, actorID, organizationID, analysisRunID); err != nil {
		return nil, err
	}
	return s.Findings.ListByAnalysisRun(ctx, organizationID, analysisRunID, limit)
}

func (s *Service) ListMatchesForActor(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.RecallMatchRecord, error) {
	if err := s.authorizeRead(ctx, actorID, organizationID, analysisRunID); err != nil {
		return nil, err
	}
	return s.Matches.ListByAnalysisRun(ctx, organizationID, analysisRunID, limit)
}

func (s *Service) authorizeRead(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID) error {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return ErrForbidden
	}
	if _, err := s.Runs.FindByID(ctx, organizationID, analysisRunID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) loadScopedEvidence(ctx context.Context, input AnalyzeInput) ([]domain.Evidence, []string, error) {
	issues := make([]string, 0)
	loaded := make([]domain.Evidence, 0, len(input.EvidenceIDs))
	for _, raw := range input.EvidenceIDs {
		id, err := primitive.ObjectIDFromHex(strings.TrimSpace(raw))
		if err != nil {
			issues = appendUnique(issues, "INVALID_EVIDENCE_ID")
			continue
		}
		ev, err := s.Evidences.GetEvidence(ctx, input.OrganizationID, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				foreign, ferr := s.Evidences.Evidences.ExistsInOtherOrg(ctx, input.OrganizationID, id)
				if ferr != nil {
					return nil, nil, ferr
				}
				if foreign {
					issues = appendUnique(issues, "WRONG_TENANT")
					continue
				}
				issues = appendUnique(issues, "INVALID_EVIDENCE_ID")
				continue
			}
			return nil, nil, err
		}
		if ev.AnalysisRunID != input.AnalysisRunID {
			issues = appendUnique(issues, "WRONG_ANALYSIS_RUN")
			continue
		}
		loaded = append(loaded, *ev)
	}
	return loaded, issues, nil
}

func (s *Service) collectRecalls(ctx context.Context, identity recall.Identity, recallIDs []string) ([]recall.Record, []string) {
	wanted := map[string]bool{}
	for _, id := range recallIDs {
		if n := strings.ToUpper(strings.TrimSpace(id)); n != "" {
			wanted[n] = true
		}
	}
	out := make([]recall.Record, 0)
	issues := make([]string, 0)
	for _, adapter := range s.Adapters {
		if adapter == nil {
			continue
		}
		items, err := adapter.SearchRecalls(ctx, identity)
		if err != nil {
			issues = appendUnique(issues, "SOURCE_UNAVAILABLE_"+adapter.Source())
			continue
		}
		for _, item := range items {
			if len(wanted) > 0 && !wanted[strings.ToUpper(item.SourceRecordID)] {
				continue
			}
			out = append(out, item)
		}
	}
	return out, issues
}

func (s *Service) persistMatch(ctx context.Context, input AnalyzeInput, rec recall.Record, match recall.MatchResult) error {
	return s.Matches.Create(ctx, &domain.RecallMatchRecord{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.AnalysisRunID,
		ProductID:      input.ProductID,
		Source:         rec.Source,
		SourceRecordID: rec.SourceRecordID,
		Matched:        match.Matched,
		Confidence:     match.Confidence,
		Method:         match.Method,
		Reference:      rec.Reference,
		RequiresReview: match.RequiresReview || !recall.Confirmed(match),
		Hazard:         rec.Hazard,
		Title:          rec.Title,
	})
}

func (s *Service) persistFinding(ctx context.Context, input AnalyzeInput, ftype domain.SafetyFindingType, severity domain.SafetySeverity, confidence float64, evidenceIDs []primitive.ObjectID, source, rationale string) (*domain.SafetyFinding, error) {
	if (severity == domain.SafetySeverityCritical || severity == domain.SafetySeverityHigh) && len(evidenceIDs) == 0 {
		ftype = domain.SafetyInsufficientEvidence
		severity = domain.SafetySeverityLow
		rationale = "INSUFFICIENT_EVIDENCE"
	}
	finding := &domain.SafetyFinding{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.AnalysisRunID,
		ProductID:      input.ProductID,
		Type:           ftype,
		Severity:       severity,
		Confidence:     confidence,
		EvidenceIDs:    evidenceIDs,
		Rationale:      rationale,
		Source:         source,
	}
	if err := s.Findings.Create(ctx, finding); err != nil {
		return nil, err
	}
	return finding, nil
}

func (s *Service) persistInsufficient(ctx context.Context, input AnalyzeInput) error {
	_, err := s.persistFinding(ctx, input, domain.SafetyInsufficientEvidence, domain.SafetySeverityLow, 0, nil, "ANALYZER", "INSUFFICIENT_EVIDENCE")
	return err
}

func classifyEvidence(ev domain.Evidence) domain.SafetyFindingType {
	if ev.Metadata != nil {
		if raw, ok := ev.Metadata["safetyType"].(string); ok && domain.ValidSafetyFindingType(raw) && raw != string(domain.SafetyInsufficientEvidence) {
			return domain.SafetyFindingType(raw)
		}
	}
	blob := strings.ToLower(ev.Claim + " " + ev.Snippet)
	switch {
	case strings.Contains(blob, "boğul") || strings.Contains(blob, "chok"):
		return domain.SafetyChoking
	case strings.Contains(blob, "recall"):
		return domain.SafetyRecall
	default:
		return ""
	}
}

func toFindingOutput(f *domain.SafetyFinding) FindingOutput {
	ids := make([]string, 0, len(f.EvidenceIDs))
	for _, id := range f.EvidenceIDs {
		ids = append(ids, id.Hex())
	}
	return FindingOutput{
		Type:        string(f.Type),
		Severity:    string(f.Severity),
		Confidence:  f.Confidence,
		EvidenceIDs: ids,
		Rationale:   f.Rationale,
	}
}

func appendUnique(items []string, issue string) []string {
	for _, existing := range items {
		if existing == issue {
			return items
		}
	}
	return append(items, issue)
}
