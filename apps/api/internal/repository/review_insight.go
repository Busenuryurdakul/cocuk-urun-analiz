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

type ReviewInsightRepository struct {
	col *mongo.Collection
}

func NewReviewInsightRepository(db *mongo.Database) *ReviewInsightRepository {
	return &ReviewInsightRepository{col: db.Collection("review_insights")}
}

func (r *ReviewInsightRepository) Create(ctx context.Context, rec *domain.ReviewInsightRecord) error {
	rec.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, rec)
	if err != nil {
		return err
	}
	rec.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ReviewInsightRepository) LatestByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID) (*domain.ReviewInsightRecord, error) {
	var rec domain.ReviewInsightRecord
	err := r.col.FindOne(ctx, bson.M{"organizationId": organizationID, "analysisRunId": analysisRunID}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&rec)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}
