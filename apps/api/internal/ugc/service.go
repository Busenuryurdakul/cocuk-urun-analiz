package ugc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidNarrative = errors.New("narrative required")
)

type Service struct {
	Experiences *repository.UserExperienceRepository
	Products    *repository.ProductRepository
	Consent     *compliance.ConsentService
	Security    *repository.SecurityEventRepository
	Tenant      *tenant.Guard
}

type CreateInput struct {
	OrganizationID    primitive.ObjectID
	ActorID           primitive.ObjectID
	ProductID         primitive.ObjectID
	UsageStatus       domain.UsageStatus
	SatisfactionLevel domain.SatisfactionLevel
	Rating            *int
	IssueType         *domain.IssueType
	Narrative         string
	MarketplaceURL    string
}

type UpdateInput struct {
	OrganizationID    primitive.ObjectID
	ActorID           primitive.ObjectID
	ExperienceID      primitive.ObjectID
	UsageStatus       domain.UsageStatus
	SatisfactionLevel domain.SatisfactionLevel
	Rating            *int
	IssueType         *domain.IssueType
	Narrative         string
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.UserExperience, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanSubmitUGC(role) {
		return nil, ErrForbidden
	}

	fields := map[string]string{
		"narrative":         input.Narrative,
		"usageStatus":       string(input.UsageStatus),
		"satisfactionLevel": string(input.SatisfactionLevel),
	}
	if err := compliance.ValidateMinimization("create_user_experience", fields); err != nil {
		return nil, err
	}

	narrative := normalize.NormalizeReviewText(input.Narrative)
	if narrative == "" {
		return nil, ErrInvalidNarrative
	}

	if _, err := s.Products.FindByID(ctx, input.OrganizationID, input.ProductID); err != nil {
		return nil, err
	}

	orgID := input.OrganizationID
	if err := s.Consent.RequireConsent(ctx, input.ActorID, &orgID, domain.ConsentPurposeDataProcessing); err != nil {
		return nil, err
	}
	consent, err := s.Consent.Current(ctx, input.ActorID, &orgID, domain.ConsentPurposeDataProcessing)
	if err != nil {
		return nil, err
	}

	eligibility := dataset.EvaluateUGCEligibility(domain.ModerationPending, domain.QualityPending, domain.PIIClean)

	ux := &domain.UserExperience{
		OrganizationID:     input.OrganizationID,
		ProductID:          input.ProductID,
		UserID:             input.ActorID,
		UsageStatus:        input.UsageStatus,
		SatisfactionLevel:  input.SatisfactionLevel,
		Rating:             input.Rating,
		IssueType:          input.IssueType,
		Narrative:          narrative,
		MarketplaceURL:     strings.TrimSpace(input.MarketplaceURL),
		SourceType:         domain.SourceTypeMiyunaUGC,
		ConsentRecordID:    consent.ID,
		PIIStatus:          domain.PIIClean,
		ModerationStatus:   domain.ModerationPending,
		QualityStatus:      domain.QualityPending,
		DatasetEligibility: eligibility,
	}
	if err := s.Experiences.Create(ctx, ux); err != nil {
		return nil, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventUGCCreated,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"experienceId": ux.ID.Hex()},
	})
	return ux, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (*domain.UserExperience, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanManageUGC(role) {
		return nil, ErrForbidden
	}

	ux, err := s.Experiences.FindByID(ctx, input.OrganizationID, input.ExperienceID)
	if err != nil {
		return nil, err
	}
	if ux.UserID != input.ActorID && role != domain.RoleOwner && role != domain.RoleAdmin {
		return nil, ErrForbidden
	}

	narrative := normalize.NormalizeReviewText(input.Narrative)
	if narrative == "" {
		return nil, ErrInvalidNarrative
	}

	ux.UsageStatus = input.UsageStatus
	ux.SatisfactionLevel = input.SatisfactionLevel
	ux.Rating = input.Rating
	ux.IssueType = input.IssueType
	ux.Narrative = narrative
	ux.ModerationStatus = domain.ModerationPending
	ux.DatasetEligibility = dataset.EvaluateUGCEligibility(ux.ModerationStatus, ux.QualityStatus, ux.PIIStatus)

	if err := s.Experiences.Update(ctx, input.OrganizationID, ux); err != nil {
		return nil, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventUGCUpdated,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"experienceId": ux.ID.Hex()},
	})
	return ux, nil
}

func (s *Service) Delete(ctx context.Context, actorID, organizationID, experienceID primitive.ObjectID) error {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return err
	}
	if !rbac.CanManageUGC(role) {
		return ErrForbidden
	}

	ux, err := s.Experiences.FindByID(ctx, organizationID, experienceID)
	if err != nil {
		return err
	}
	if ux.UserID != actorID && role != domain.RoleOwner && role != domain.RoleAdmin {
		return ErrForbidden
	}

	if err := s.Experiences.Delete(ctx, organizationID, experienceID); err != nil {
		return err
	}

	uid := actorID
	oid := organizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventUGCDeleted,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"experienceId": experienceID.Hex()},
	})
	return nil
}

type ModerateInput struct {
	OrganizationID   primitive.ObjectID
	ActorID          primitive.ObjectID
	ExperienceID     primitive.ObjectID
	ModerationStatus domain.ModerationStatus
	QualityStatus    *domain.QualityStatus
}

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (*domain.UserExperience, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanModerateUGC(role) {
		return nil, ErrForbidden
	}
	if input.ModerationStatus != domain.ModerationApproved && input.ModerationStatus != domain.ModerationRejected {
		return nil, fmt.Errorf("invalid moderation status")
	}

	ux, err := s.Experiences.FindByID(ctx, input.OrganizationID, input.ExperienceID)
	if err != nil {
		return nil, err
	}

	ux.ModerationStatus = input.ModerationStatus
	if input.QualityStatus != nil {
		ux.QualityStatus = *input.QualityStatus
	} else if input.ModerationStatus == domain.ModerationApproved {
		ux.QualityStatus = domain.QualityApproved
	} else {
		ux.QualityStatus = domain.QualityRejected
	}
	ux.DatasetEligibility = dataset.EvaluateUGCEligibility(ux.ModerationStatus, ux.QualityStatus, ux.PIIStatus)

	if err := s.Experiences.Update(ctx, input.OrganizationID, ux); err != nil {
		return nil, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventUGCUpdated,
		Severity:       domain.SeverityInfo,
		Details: map[string]string{
			"experienceId": ux.ID.Hex(),
			"moderation":   string(ux.ModerationStatus),
		},
	})
	return ux, nil
}

func (s *Service) ListByProduct(ctx context.Context, actorID, organizationID, productID primitive.ObjectID) ([]domain.UserExperience, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadUGC(role) {
		return nil, ErrForbidden
	}
	return s.Experiences.ListByProduct(ctx, organizationID, productID)
}

// ValidateCreateInput exposes validation for tests.
func ValidateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.Narrative) == "" {
		return fmt.Errorf("%w", ErrInvalidNarrative)
	}
	return compliance.ValidateMinimization("create_user_experience", map[string]string{
		"narrative":         input.Narrative,
		"usageStatus":       string(input.UsageStatus),
		"satisfactionLevel": string(input.SatisfactionLevel),
	})
}
