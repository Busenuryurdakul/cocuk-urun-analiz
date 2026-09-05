package graph

import (
	"errors"
	"fmt"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/fetch"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/ugc"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func mapPhase4Error(err error) error {
	switch {
	case errors.Is(err, product.ErrForbidden), errors.Is(err, ugc.ErrForbidden), errors.Is(err, marketplace.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, ugc.ErrInvalidNarrative):
		return gqlError("INVALID_INPUT", err)
	case errors.Is(err, fetch.ErrHostNotAllowed), errors.Is(err, fetch.ErrInvalidURL), errors.Is(err, fetch.ErrSchemeNotAllowed):
		return gqlError("FETCH_POLICY_VIOLATION", err)
	case errors.Is(err, marketplace.ErrInvalidSource):
		return gqlError("INVALID_SOURCE", err)
	default:
		return mapAuthError(err)
	}
}

func toModelProductField(meta domain.ProductFieldMeta) *model.ProductFieldMeta {
	var value *string
	if meta.Value != nil {
		s := fmt.Sprint(meta.Value)
		value = &s
	}
	return &model.ProductFieldMeta{
		Value:          value,
		Missing:        meta.Missing,
		Source:         strPtr(meta.Source),
		SourceRecordID: strPtr(meta.SourceRecordID),
	}
}

func toModelProduct(p *domain.Product) *model.Product {
	return &model.Product{
		ID:             p.ID.Hex(),
		OrganizationID: p.OrganizationID.Hex(),
		Name:           toModelProductField(p.Name),
		Brand:          toModelProductField(p.Brand),
		Category:       toModelProductField(p.Category),
		Description:    toModelProductField(p.Description),
		Sku:            toModelProductField(p.SKU),
		CreatedAt:      p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toModelUserExperience(ux *domain.UserExperience) *model.UserExperience {
	out := &model.UserExperience{
		ID:                 ux.ID.Hex(),
		OrganizationID:     ux.OrganizationID.Hex(),
		ProductID:          ux.ProductID.Hex(),
		UserID:             ux.UserID.Hex(),
		UsageStatus:        model.UsageStatus(ux.UsageStatus),
		SatisfactionLevel:  model.SatisfactionLevel(ux.SatisfactionLevel),
		Rating:             ux.Rating,
		Narrative:          ux.Narrative,
		ModerationStatus:   model.ModerationStatus(ux.ModerationStatus),
		QualityStatus:      model.QualityStatus(ux.QualityStatus),
		DatasetEligibility: model.DatasetEligibility(ux.DatasetEligibility),
		CreatedAt:          ux.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          ux.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if ux.IssueType != nil {
		v := model.IssueType(*ux.IssueType)
		out.IssueType = &v
	}
	return out
}

func toModelImportRun(run *domain.MarketplaceImportRun) *model.MarketplaceImportRun {
	out := &model.MarketplaceImportRun{
		ID:              run.ID.Hex(),
		OrganizationID:  run.OrganizationID.Hex(),
		Source:          model.MarketplaceSource(run.Source),
		SourceURL:       strPtr(run.SourceURL),
		AccessMode:      model.AccessMode(run.AccessMode),
		Status:          model.MarketplaceImportStatus(run.Status),
		RecordsSeen:     run.RecordsSeen,
		RecordsAccepted: run.RecordsAccepted,
		RecordsRejected: run.RecordsRejected,
		ErrorCode:       strPtr(run.ErrorCode),
		ErrorMessage:    strPtr(run.ErrorMessage),
		CreatedAt:       run.CreatedAt.UTC().Format(time.RFC3339),
	}
	return out
}

func toModelDatasetVersion(v *domain.DatasetVersion) *model.DatasetVersion {
	return &model.DatasetVersion{
		ID:                v.ID.Hex(),
		OrganizationID:    v.OrganizationID.Hex(),
		Version:           v.Version,
		Status:            model.DatasetVersionStatus(v.Status),
		NormalizerVersion: v.NormalizerVersion,
		ContentHash:       strPtr(v.ContentHash),
		CreatedAt:         v.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toModelMarketplaceReview(rev *domain.MarketplaceReview) *model.MarketplaceReview {
	out := &model.MarketplaceReview{
		ID:                 rev.ID.Hex(),
		OrganizationID:     rev.OrganizationID.Hex(),
		ProductID:          rev.ProductID.Hex(),
		Source:             model.MarketplaceSource(rev.Source),
		Rating:             rev.Rating,
		ReviewText:         rev.ReviewText,
		Language:           string(rev.Language),
		ModerationStatus:   model.ModerationStatus(rev.ModerationStatus),
		DatasetEligibility: model.DatasetEligibility(rev.DatasetEligibility),
		CreatedAt:          rev.CreatedAt.UTC().Format(time.RFC3339),
	}
	if rev.ReviewDate != nil {
		s := rev.ReviewDate.UTC().Format(time.RFC3339)
		out.ReviewDate = &s
	}
	return out
}

func toModelEligibilitySummary(s *dataset.EligibilitySummary) *model.DatasetEligibilitySummary {
	elig := make([]*model.EligibilityCount, 0, len(s.EligibilityDistribution))
	for k, v := range s.EligibilityDistribution {
		elig = append(elig, &model.EligibilityCount{
			Eligibility: model.DatasetEligibility(k),
			Count:       v,
		})
	}
	sources := make([]*model.SourceCount, 0, len(s.SourceDistribution))
	for k, v := range s.SourceDistribution {
		sources = append(sources, &model.SourceCount{
			RecordType: k,
			Count:      v,
		})
	}
	return &model.DatasetEligibilitySummary{
		OrganizationID:          s.OrganizationID.Hex(),
		TotalRecords:            s.TotalRecords,
		UgcCount:                s.UGCCount,
		MarketplaceCount:        s.MarketplaceCount,
		EligibilityDistribution: elig,
		SourceDistribution:      sources,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseOrgProductIDs(organizationID, secondID string) (primitive.ObjectID, primitive.ObjectID, error) {
	orgID, err := parseObjectID(organizationID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, err
	}
	otherID, err := parseObjectID(secondID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, err
	}
	return orgID, otherID, nil
}

func limitOrDefault(limit *int, fallback int) int {
	if limit == nil {
		return fallback
	}
	return *limit
}

func derefIssueType(v *model.IssueType) *domain.IssueType {
	if v == nil {
		return nil
	}
	out := domain.IssueType(*v)
	return &out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
