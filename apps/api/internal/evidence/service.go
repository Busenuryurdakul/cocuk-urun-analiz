package evidence

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidInput       = errors.New("invalid evidence input")
	ErrForeignAnalysisRun = errors.New("analysis run not in organization")
	ErrForeignProduct     = errors.New("product not in organization")
	ErrProductRunMismatch = errors.New("product does not belong to analysis run")
)

type Service struct {
	Evidences   *repository.EvidenceRepository
	Validations *repository.EvidenceClaimValidationRepository
	Runs        *repository.AnalysisRunRepository
	Products    *repository.ProductRepository
	Tenant      *tenant.Guard
}

type CreateEvidenceInput struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	ProductID      primitive.ObjectID
	ClaimID        string
	Source         string
	SourceType     string
	Claim          string
	Snippet        string
	Reference      string
	Reliability    float64
	Freshness      float64
	RetrievedAt    time.Time
	Metadata       map[string]any
}

func (s *Service) CreateEvidence(ctx context.Context, input CreateEvidenceInput) (*domain.Evidence, error) {
	if input.OrganizationID.IsZero() || input.AnalysisRunID.IsZero() || input.ProductID.IsZero() {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.SourceType) == "" || strings.TrimSpace(input.Claim) == "" {
		return nil, ErrInvalidInput
	}
	if err := validateScore(input.Reliability); err != nil {
		return nil, err
	}
	if err := validateScore(input.Freshness); err != nil {
		return nil, err
	}

	run, err := s.Runs.FindByID(ctx, input.OrganizationID, input.AnalysisRunID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrForeignAnalysisRun
		}
		return nil, err
	}
	product, err := s.Products.FindByID(ctx, input.OrganizationID, input.ProductID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrForeignProduct
		}
		return nil, err
	}
	if run.ProductID != product.ID {
		return nil, ErrProductRunMismatch
	}

	retrievedAt := input.RetrievedAt
	if retrievedAt.IsZero() {
		retrievedAt = time.Now().UTC()
	}
	ev := &domain.Evidence{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.AnalysisRunID,
		ProductID:      input.ProductID,
		ClaimID:        strings.TrimSpace(input.ClaimID),
		Source:         strings.TrimSpace(input.Source),
		SourceType:     strings.TrimSpace(input.SourceType),
		Claim:          strings.TrimSpace(input.Claim),
		Snippet:        strings.TrimSpace(input.Snippet),
		Reference:      strings.TrimSpace(input.Reference),
		Reliability:    input.Reliability,
		Freshness:      input.Freshness,
		RetrievedAt:    retrievedAt,
		Metadata:       input.Metadata,
	}
	if err := s.Evidences.Create(ctx, ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (s *Service) GetEvidence(ctx context.Context, organizationID, evidenceID primitive.ObjectID) (*domain.Evidence, error) {
	if organizationID.IsZero() || evidenceID.IsZero() {
		return nil, ErrInvalidInput
	}
	return s.Evidences.FindByID(ctx, organizationID, evidenceID)
}

func (s *Service) ListByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.Evidence, error) {
	if organizationID.IsZero() || analysisRunID.IsZero() {
		return nil, ErrInvalidInput
	}
	if _, err := s.Runs.FindByID(ctx, organizationID, analysisRunID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []domain.Evidence{}, nil
		}
		return nil, err
	}
	return s.Evidences.ListByAnalysisRun(ctx, organizationID, analysisRunID, limit)
}

func (s *Service) ListByProduct(ctx context.Context, organizationID, productID primitive.ObjectID, limit int) ([]domain.Evidence, error) {
	if organizationID.IsZero() || productID.IsZero() {
		return nil, ErrInvalidInput
	}
	if _, err := s.Products.FindByID(ctx, organizationID, productID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []domain.Evidence{}, nil
		}
		return nil, err
	}
	return s.Evidences.ListByProduct(ctx, organizationID, productID, limit)
}

func (s *Service) ListByClaim(ctx context.Context, organizationID primitive.ObjectID, claimID string, limit int) ([]domain.Evidence, error) {
	if organizationID.IsZero() || strings.TrimSpace(claimID) == "" {
		return nil, ErrInvalidInput
	}
	return s.Evidences.ListByClaim(ctx, organizationID, strings.TrimSpace(claimID), limit)
}

func (s *Service) ListAnalysisEvidenceForActor(ctx context.Context, actorID, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.Evidence, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadAnalysisRun(role) {
		return nil, ErrForbidden
	}
	return s.ListByAnalysisRun(ctx, organizationID, analysisRunID, limit)
}

func validateScore(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
		return ErrInvalidInput
	}
	return nil
}
