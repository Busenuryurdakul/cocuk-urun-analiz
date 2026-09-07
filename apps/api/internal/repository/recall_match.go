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

type RecallMatchRepository struct {
	col *mongo.Collection
}

func NewRecallMatchRepository(db *mongo.Database) *RecallMatchRepository {
	return &RecallMatchRepository{col: db.Collection("recall_matches")}
}

func (r *RecallMatchRepository) Create(ctx context.Context, rec *domain.RecallMatchRecord) error {
	rec.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, rec)
	if err != nil {
		return err
	}
	rec.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RecallMatchRepository) ListByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.RecallMatchRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID, "analysisRunId": analysisRunID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var items []domain.RecallMatchRecord
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.RecallMatchRecord{}
	}
	return items, nil
}
