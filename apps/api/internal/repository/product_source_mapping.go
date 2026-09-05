package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductSourceMappingRepository struct {
	col *mongo.Collection
}

func NewProductSourceMappingRepository(db *mongo.Database) *ProductSourceMappingRepository {
	return &ProductSourceMappingRepository{col: db.Collection("product_source_mappings")}
}

func (r *ProductSourceMappingRepository) Create(ctx context.Context, mapping *domain.ProductSourceMapping) error {
	now := time.Now().UTC()
	mapping.CreatedAt = now
	mapping.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, mapping)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	mapping.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProductSourceMappingRepository) FindBySourceProduct(ctx context.Context, organizationID primitive.ObjectID, source domain.MarketplaceSource, sourceProductID string) (*domain.ProductSourceMapping, error) {
	var mapping domain.ProductSourceMapping
	err := r.col.FindOne(ctx, bson.M{
		"organizationId":  organizationID,
		"source":          source,
		"sourceProductId": sourceProductID,
	}).Decode(&mapping)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &mapping, nil
}

func (r *ProductSourceMappingRepository) FindByProductID(ctx context.Context, organizationID, productID primitive.ObjectID) ([]domain.ProductSourceMapping, error) {
	cur, err := r.col.Find(ctx, bson.M{
		"organizationId": organizationID,
		"productId":      productID,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var mappings []domain.ProductSourceMapping
	if err := cur.All(ctx, &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}
