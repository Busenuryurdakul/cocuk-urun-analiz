package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MarketplaceReviewRepository struct {
	col *mongo.Collection
}

func NewMarketplaceReviewRepository(db *mongo.Database) *MarketplaceReviewRepository {
	return &MarketplaceReviewRepository{col: db.Collection("marketplace_reviews")}
}

func (r *MarketplaceReviewRepository) Create(ctx context.Context, review *domain.MarketplaceReview) error {
	review.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, review)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	review.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MarketplaceReviewRepository) FindByFingerprint(ctx context.Context, organizationID primitive.ObjectID, fingerprint string) (*domain.MarketplaceReview, error) {
	var review domain.MarketplaceReview
	err := r.col.FindOne(ctx, bson.M{
		"organizationId": organizationID,
		"fingerprint":    fingerprint,
	}).Decode(&review)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *MarketplaceReviewRepository) ListByProduct(ctx context.Context, organizationID, productID primitive.ObjectID) ([]domain.MarketplaceReview, error) {
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID, "productId": productID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var reviews []domain.MarketplaceReview
	if err := cur.All(ctx, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}
