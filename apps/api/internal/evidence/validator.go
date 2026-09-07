package evidence

import (
	"context"
	"errors"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ValidateClaimInput struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	ClaimID        string
	ClaimText      string
	EvidenceIDs    []string
}

type ValidateClaimResult struct {
	ClaimID     string
	Status      domain.EvidenceSupportStatus
	EvidenceIDs []string
	Issues      []string
}

func (s *Service) ValidateClaim(ctx context.Context, input ValidateClaimInput) (*ValidateClaimResult, error) {
	if input.OrganizationID.IsZero() || input.AnalysisRunID.IsZero() {
		return nil, ErrInvalidInput
	}
	claimID := strings.TrimSpace(input.ClaimID)
	claimText := strings.TrimSpace(input.ClaimText)
	if claimID == "" || claimText == "" {
		return nil, ErrInvalidInput
	}
	if _, err := s.Runs.FindByID(ctx, input.OrganizationID, input.AnalysisRunID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrForeignAnalysisRun
		}
		return nil, err
	}

	issues := make([]string, 0)
	accepted := make([]primitive.ObjectID, 0)
	acceptedHex := make([]string, 0)
	contradicted := false

	if len(input.EvidenceIDs) == 0 {
		issues = appendIssue(issues, domain.EvidenceIssueNoEvidence)
		return s.persistValidation(ctx, input, claimID, claimText, domain.EvidenceUnsupported, nil, issues)
	}

	for _, rawID := range input.EvidenceIDs {
		id, err := primitive.ObjectIDFromHex(strings.TrimSpace(rawID))
		if err != nil {
			issues = appendIssue(issues, domain.EvidenceIssueInvalidEvidenceID)
			continue
		}
		ev, err := s.Evidences.FindByID(ctx, input.OrganizationID, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				foreign, ferr := s.Evidences.ExistsInOtherOrg(ctx, input.OrganizationID, id)
				if ferr != nil {
					return nil, ferr
				}
				if foreign {
					issues = appendIssue(issues, domain.EvidenceIssueWrongTenant)
					continue
				}
				issues = appendIssue(issues, domain.EvidenceIssueInvalidEvidenceID)
				continue
			}
			return nil, err
		}
		if ev.AnalysisRunID != input.AnalysisRunID {
			issues = appendIssue(issues, domain.EvidenceIssueWrongAnalysisRun)
			continue
		}
		if ev.ClaimID != "" && ev.ClaimID != claimID {
			issues = appendIssue(issues, domain.EvidenceIssueClaimMismatch)
			continue
		}
		if strings.TrimSpace(ev.Source) == "" {
			issues = appendIssue(issues, domain.EvidenceIssueMissingSource)
			continue
		}
		if referenceRequired(ev.SourceType) && strings.TrimSpace(ev.Reference) == "" {
			issues = appendIssue(issues, domain.EvidenceIssueMissingReference)
			continue
		}
		if err := validateScore(ev.Reliability); err != nil {
			issues = appendIssue(issues, domain.EvidenceIssueInvalidReliability)
			continue
		}
		if err := validateScore(ev.Freshness); err != nil {
			issues = appendIssue(issues, domain.EvidenceIssueInvalidFreshness)
			continue
		}
		if isContradictory(ev) {
			contradicted = true
			issues = appendIssue(issues, domain.EvidenceIssueContradictory)
			accepted = append(accepted, ev.ID)
			acceptedHex = append(acceptedHex, ev.ID.Hex())
			continue
		}
		accepted = append(accepted, ev.ID)
		acceptedHex = append(acceptedHex, ev.ID.Hex())
	}

	status := domain.EvidenceUnsupported
	switch {
	case contradicted:
		status = domain.EvidenceContradicted
	case len(accepted) == 0:
		status = domain.EvidenceUnsupported
	case len(issues) == 0 && len(accepted) == len(input.EvidenceIDs):
		status = domain.EvidenceSupported
	case len(accepted) > 0:
		status = domain.EvidencePartiallySupported
	}

	return s.persistValidation(ctx, input, claimID, claimText, status, accepted, issues)
}

func (s *Service) persistValidation(
	ctx context.Context,
	input ValidateClaimInput,
	claimID, claimText string,
	status domain.EvidenceSupportStatus,
	accepted []primitive.ObjectID,
	issues []string,
) (*ValidateClaimResult, error) {
	if accepted == nil {
		accepted = []primitive.ObjectID{}
	}
	if issues == nil {
		issues = []string{}
	}
	record := &domain.EvidenceClaimValidation{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.AnalysisRunID,
		ClaimID:        claimID,
		ClaimText:      claimText,
		EvidenceIDs:    accepted,
		SupportStatus:  status,
		Issues:         issues,
	}
	if err := s.Validations.Upsert(ctx, record); err != nil {
		return nil, err
	}
	hexIDs := make([]string, 0, len(accepted))
	for _, id := range accepted {
		hexIDs = append(hexIDs, id.Hex())
	}
	return &ValidateClaimResult{
		ClaimID:     claimID,
		Status:      status,
		EvidenceIDs: hexIDs,
		Issues:      issues,
	}, nil
}

func referenceRequired(sourceType string) bool {
	switch strings.ToUpper(strings.TrimSpace(sourceType)) {
	case "URL", "MARKETPLACE_REVIEW", "CSV", "JSON":
		return true
	default:
		return false
	}
}

func isContradictory(ev *domain.Evidence) bool {
	if ev == nil || ev.Metadata == nil {
		return false
	}
	if flag, ok := ev.Metadata["contradictsClaim"].(bool); ok && flag {
		return true
	}
	if stance, ok := ev.Metadata["stance"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(stance)) {
		case "contradict", "contradicted", "contradicts":
			return true
		}
	}
	return false
}

func appendIssue(issues []string, issue string) []string {
	for _, existing := range issues {
		if existing == issue {
			return issues
		}
	}
	return append(issues, issue)
}
