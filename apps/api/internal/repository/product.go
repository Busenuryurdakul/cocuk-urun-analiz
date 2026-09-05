package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepository struct {
	col *mongo.Collection
}

func NewProductRepository(db *mongo.Database) *ProductRepository {
	return &ProductRepository{col: db.Collection("products")}
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, product)
	if err != nil {
		return err
	}
	product.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, organizationID, productID primitive.ObjectID) (*domain.Product, error) {
	var product domain.Product
	err := r.col.FindOne(ctx, bson.M{
		"_id":            productID,
		"organizationId": organizationID,
	}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) ListByOrg(ctx context.Context, organizationID primitive.ObjectID, limit int) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 50
	}
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID}, mongoOptionsLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var products []domain.Product
	if err := cur.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func mongoOptionsLimit(limit int) *options.FindOptions {
	return options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "createdAt", Value: -1}})
}
