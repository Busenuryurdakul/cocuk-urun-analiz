package agent

import "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"

type ResolvedInput struct {
	Product                *domain.Product
	MarketplaceReviews     []domain.MarketplaceReview
	UserExperiences        []domain.UserExperience
	MarketplaceReviewCount int
	UGCCount               int
}
