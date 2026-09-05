package agent

import (
	"context"
	"errors"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InputResolver struct {
	Products   *repository.ProductRepository
	Reviews    *repository.MarketplaceReviewRepository
	UX         *repository.UserExperienceRepository
	ImportRuns *repository.MarketplaceImportRunRepository
}

func (r *InputResolver) Resolve(ctx context.Context, organizationID, productID primitive.ObjectID, importRunID *primitive.ObjectID) (*ResolvedInput, error) {
	product, err := r.Products.FindByID(ctx, organizationID, productID)
	if err != nil {
		return nil, err
	}

	if importRunID != nil {
		run, err := r.ImportRuns.FindByID(ctx, organizationID, *importRunID)
		if err != nil {
			return nil, err
		}
		if run.ProductID != nil && *run.ProductID != productID {
			return nil, ErrInvalidInput
		}
	}

	reviews, err := r.Reviews.ListByProduct(ctx, organizationID, productID)
	if err != nil {
		return nil, err
	}
	uxList, err := r.UX.ListByProduct(ctx, organizationID, productID)
	if err != nil {
		return nil, err
	}

	approvedReviews := filterApprovedReviews(reviews)
	approvedUX := filterApprovedUX(uxList)

	return &ResolvedInput{
		Product:                product,
		MarketplaceReviews:     approvedReviews,
		UserExperiences:        approvedUX,
		MarketplaceReviewCount: len(approvedReviews),
		UGCCount:               len(approvedUX),
	}, nil
}

func filterApprovedReviews(reviews []domain.MarketplaceReview) []domain.MarketplaceReview {
	out := make([]domain.MarketplaceReview, 0, len(reviews))
	for _, rv := range reviews {
		if rv.ModerationStatus == domain.ModerationApproved {
			out = append(out, rv)
		}
	}
	return out
}

func filterApprovedUX(list []domain.UserExperience) []domain.UserExperience {
	out := make([]domain.UserExperience, 0, len(list))
	for _, ux := range list {
		if ux.ModerationStatus == domain.ModerationApproved {
			out = append(out, ux)
		}
	}
	return out
}

func (r *InputResolver) ValidateProvenance(ctx context.Context, organizationID primitive.ObjectID, importRunID *primitive.ObjectID, productID primitive.ObjectID) error {
	if importRunID == nil {
		return nil
	}
	run, err := r.ImportRuns.FindByID(ctx, organizationID, *importRunID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidInput
		}
		return err
	}
	if run.ProductID != nil && *run.ProductID != productID {
		return ErrInvalidInput
	}
	return nil
}
