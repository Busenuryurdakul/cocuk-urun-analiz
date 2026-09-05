package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MarketplaceImportRunRepository struct {
	col *mongo.Collection
}

func NewMarketplaceImportRunRepository(db *mongo.Database) *MarketplaceImportRunRepository {
	return &MarketplaceImportRunRepository{col: db.Collection("marketplace_import_runs")}
}

func (r *MarketplaceImportRunRepository) Create(ctx context.Context, run *domain.MarketplaceImportRun) error {
	run.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, run)
	if err != nil {
		return err
	}
	run.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MarketplaceImportRunRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.MarketplaceImportRun, error) {
	var run domain.MarketplaceImportRun
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": organizationID}).Decode(&run)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *MarketplaceImportRunRepository) Update(ctx context.Context, organizationID primitive.ObjectID, run *domain.MarketplaceImportRun) error {
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": run.ID, "organizationId": organizationID}, bson.M{"$set": run})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MarketplaceImportRunRepository) ListByOrg(ctx context.Context, organizationID primitive.ObjectID, limit int) ([]domain.MarketplaceImportRun, error) {
	if limit <= 0 {
		limit = 20
	}
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID}, mongoOptionsLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var runs []domain.MarketplaceImportRun
	if err := cur.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}
