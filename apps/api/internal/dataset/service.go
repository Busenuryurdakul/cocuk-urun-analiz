package dataset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrForbidden = errors.New("forbidden")

type Service struct {
	Records  *repository.DatasetRecordRepository
	Versions *repository.DatasetVersionRepository
	Security *repository.SecurityEventRepository
	Tenant   *tenant.Guard
}

type EligibilityInput struct {
	ModerationStatus  domain.ModerationStatus
	QualityStatus     domain.QualityStatus
	PIIStatus         domain.PIIStatus
	ProvenanceStatus  domain.ProvenanceStatus
	LicenseStatus     domain.LicenseStatus
	UsageRightsStatus domain.UsageRightsStatus
	SpamStatus        domain.SpamStatus
	DuplicateStatus   domain.DuplicateStatus
}

func EvaluateEligibility(in EligibilityInput) domain.DatasetEligibility {
	if in.ModerationStatus == domain.ModerationRejected || in.QualityStatus == domain.QualityRejected {
		return domain.EligibilityRejected
	}
	if in.PIIStatus == domain.PIIQuarantined || in.SpamStatus == domain.SpamSuspect {
		return domain.EligibilityQuarantined
	}
	if in.DuplicateStatus == domain.DuplicateDetected {
		return domain.EligibilityAnalysisOnly
	}
	if in.ModerationStatus != domain.ModerationApproved || in.QualityStatus != domain.QualityApproved {
		return domain.EligibilityAnalysisOnly
	}
	if in.LicenseStatus == domain.LicenseRestricted || in.UsageRightsStatus == domain.UsageRightsRestricted {
		return domain.EligibilityEvalOnly
	}
	if in.ProvenanceStatus == domain.ProvenanceVerified &&
		in.LicenseStatus == domain.LicenseApproved &&
		in.UsageRightsStatus == domain.UsageRightsApproved &&
		in.PIIStatus == domain.PIIClean {
		return domain.EligibilityTrainingApproved
	}
	return domain.EligibilityAnalysisOnly
}

func EvaluateUGCEligibility(moderation domain.ModerationStatus, quality domain.QualityStatus, pii domain.PIIStatus) domain.DatasetEligibility {
	return EvaluateEligibility(EligibilityInput{
		ModerationStatus:  moderation,
		QualityStatus:     quality,
		PIIStatus:         pii,
		ProvenanceStatus:  domain.ProvenanceVerified,
		LicenseStatus:     domain.LicenseApproved,
		UsageRightsStatus: domain.UsageRightsApproved,
		SpamStatus:        domain.SpamClean,
		DuplicateStatus:   domain.DuplicateUnique,
	})
}

// IsTrainingEligible reports whether a record may enter training selection.
func IsTrainingEligible(eligibility domain.DatasetEligibility) bool {
	return eligibility == domain.EligibilityTrainingApproved
}

// RejectsTrainingSelection reports eligibility values excluded from training.
func RejectsTrainingSelection(eligibility domain.DatasetEligibility) bool {
	switch eligibility {
	case domain.EligibilityQuarantined,
		domain.EligibilityAnalysisOnly,
		domain.EligibilityEvalOnly,
		domain.EligibilityRejected:
		return true
	default:
		return false
	}
}
func IsHeldOut(record domain.DatasetRecord) bool {
	return record.Split != nil && *record.Split == domain.SplitHeldOutEval
}

// ExcludeHeldOutFilter returns a Mongo filter clause excluding held-out records from training queries.
func ExcludeHeldOutFilter() bson.M {
	return bson.M{
		"$or": []bson.M{
			{"split": bson.M{"$exists": false}},
			{"split": bson.M{"$ne": domain.SplitHeldOutEval}},
		},
	}
}

// TrainingEligibleFilter combines training eligibility with held-out isolation.
func TrainingEligibleFilter(organizationID primitive.ObjectID) bson.M {
	return bson.M{
		"organizationId":     organizationID,
		"datasetEligibility": domain.EligibilityTrainingApproved,
		"$and":               []bson.M{ExcludeHeldOutFilter()},
	}
}

type BuildDraftInput struct {
	OrganizationID primitive.ObjectID
	ActorID        primitive.ObjectID
	VersionLabel   string
}

func (s *Service) BuildDatasetDraft(ctx context.Context, input BuildDraftInput) (*domain.DatasetVersion, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanManageDataset(role) {
		return nil, ErrForbidden
	}

	records, err := s.Records.ListEligible(ctx, input.OrganizationID, domain.EligibilityTrainingApproved)
	if err != nil {
		return nil, err
	}

	eligibilityDist := map[string]int{}
	sourceDist := map[string]int{}
	for _, rec := range records {
		if IsHeldOut(rec) {
			continue
		}
		eligibilityDist[string(rec.DatasetEligibility)]++
		sourceDist[string(rec.RecordType)]++
	}

	hashPayload := fmt.Sprintf("%s:%d", input.OrganizationID.Hex(), len(records))
	sum := sha256.Sum256([]byte(hashPayload))
	version := &domain.DatasetVersion{
		OrganizationID:          input.OrganizationID,
		Version:                 input.VersionLabel,
		Status:                  domain.DatasetDraft,
		SourceCounts:            map[string]int{"records": len(records)},
		SourceDistribution:      sourceDist,
		EligibilityDistribution: eligibilityDist,
		NormalizerVersion:       domain.NormalizerVersionPhase4,
		PIIPolicyVersion:        "phase4-v1",
		LicensePolicyVersion:    "phase4-v1",
		ContentHash:             hex.EncodeToString(sum[:]),
		CreatedBy:               input.ActorID,
	}
	if err := s.Versions.Create(ctx, version); err != nil {
		return nil, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventDatasetDraftBuilt,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"versionId": version.ID.Hex(), "records": fmt.Sprintf("%d", len(records))},
	})
	return version, nil
}

type EligibilitySummary struct {
	OrganizationID          primitive.ObjectID
	TotalRecords            int
	UGCCount                int
	MarketplaceCount        int
	EligibilityDistribution map[string]int
	SourceDistribution      map[string]int
}

func (s *Service) EligibilitySummary(ctx context.Context, actorID, organizationID primitive.ObjectID) (*EligibilitySummary, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadDataset(role) {
		return nil, ErrForbidden
	}
	elig, err := s.Records.CountByEligibility(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	types, err := s.Records.CountByRecordType(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, c := range elig {
		total += c
	}
	return &EligibilitySummary{
		OrganizationID:          organizationID,
		TotalRecords:            total,
		UGCCount:                types[string(domain.RecordMiyunaUGC)],
		MarketplaceCount:        types[string(domain.RecordMarketplaceReview)],
		EligibilityDistribution: elig,
		SourceDistribution:      types,
	}, nil
}

func (s *Service) ListVersions(ctx context.Context, actorID, organizationID primitive.ObjectID, limit int) ([]domain.DatasetVersion, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadDataset(role) {
		return nil, ErrForbidden
	}
	return s.Versions.ListByOrg(ctx, organizationID, limit)
}
