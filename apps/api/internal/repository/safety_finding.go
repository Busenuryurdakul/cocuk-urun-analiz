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

type SafetyFindingRepository struct {
	col *mongo.Collection
}

func NewSafetyFindingRepository(db *mongo.Database) *SafetyFindingRepository {
	return &SafetyFindingRepository{col: db.Collection("safety_findings")}
}

func (r *SafetyFindingRepository) Create(ctx context.Context, finding *domain.SafetyFinding) error {
	finding.CreatedAt = time.Now().UTC()
	if finding.EvidenceIDs == nil {
		finding.EvidenceIDs = []primitive.ObjectID{}
	}
	res, err := r.col.InsertOne(ctx, finding)
	if err != nil {
		return err
	}
	finding.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *SafetyFindingRepository) FindByID(ctx context.Context, organizationID, findingID primitive.ObjectID) (*domain.SafetyFinding, error) {
	var finding domain.SafetyFinding
	err := r.col.FindOne(ctx, bson.M{"_id": findingID, "organizationId": organizationID}).Decode(&finding)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &finding, nil
}

func (r *SafetyFindingRepository) ListByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.SafetyFinding, error) {
	return r.list(ctx, bson.M{"organizationId": organizationID, "analysisRunId": analysisRunID}, limit)
}

func (r *SafetyFindingRepository) list(ctx context.Context, filter bson.M, limit int) ([]domain.SafetyFinding, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var items []domain.SafetyFinding
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.SafetyFinding{}
	}
	return items, nil
}
