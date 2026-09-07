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

type EvidenceRepository struct {
	col *mongo.Collection
}

func NewEvidenceRepository(db *mongo.Database) *EvidenceRepository {
	return &EvidenceRepository{col: db.Collection("evidences")}
}

func (r *EvidenceRepository) Create(ctx context.Context, ev *domain.Evidence) error {
	now := time.Now().UTC()
	if ev.RetrievedAt.IsZero() {
		ev.RetrievedAt = now
	}
	ev.CreatedAt = now
	ev.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, ev)
	if err != nil {
		return err
	}
	ev.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *EvidenceRepository) FindByID(ctx context.Context, organizationID, evidenceID primitive.ObjectID) (*domain.Evidence, error) {
	var ev domain.Evidence
	err := r.col.FindOne(ctx, bson.M{
		"_id":            evidenceID,
		"organizationId": organizationID,
	}).Decode(&ev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ev, nil
}

func (r *EvidenceRepository) ExistsInOtherOrg(ctx context.Context, organizationID, evidenceID primitive.ObjectID) (bool, error) {
	err := r.col.FindOne(ctx, bson.M{
		"_id":            evidenceID,
		"organizationId": bson.M{"$ne": organizationID},
	}).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *EvidenceRepository) ListByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.Evidence, error) {
	return r.list(ctx, bson.M{
		"organizationId": organizationID,
		"analysisRunId":  analysisRunID,
	}, limit)
}

func (r *EvidenceRepository) ListByProduct(ctx context.Context, organizationID, productID primitive.ObjectID, limit int) ([]domain.Evidence, error) {
	return r.list(ctx, bson.M{
		"organizationId": organizationID,
		"productId":      productID,
	}, limit)
}

func (r *EvidenceRepository) ListByClaim(ctx context.Context, organizationID primitive.ObjectID, claimID string, limit int) ([]domain.Evidence, error) {
	return r.list(ctx, bson.M{
		"organizationId": organizationID,
		"claimId":        claimID,
	}, limit)
}

func (r *EvidenceRepository) list(ctx context.Context, filter bson.M, limit int) ([]domain.Evidence, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var items []domain.Evidence
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Evidence{}
	}
	return items, nil
}
