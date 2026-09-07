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

type AnalysisRunRepository struct {
	col *mongo.Collection
}

func NewAnalysisRunRepository(db *mongo.Database) *AnalysisRunRepository {
	return &AnalysisRunRepository{col: db.Collection("analysis_runs")}
}

func (r *AnalysisRunRepository) Create(ctx context.Context, run *domain.AnalysisRun) error {
	now := time.Now().UTC()
	run.CreatedAt = now
	run.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, run)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	run.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AnalysisRunRepository) FindByID(ctx context.Context, organizationID, runID primitive.ObjectID) (*domain.AnalysisRun, error) {
	var run domain.AnalysisRun
	err := r.col.FindOne(ctx, bson.M{
		"_id":            runID,
		"organizationId": organizationID,
	}).Decode(&run)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *AnalysisRunRepository) FindByClientRequestID(ctx context.Context, organizationID primitive.ObjectID, clientRequestID string) (*domain.AnalysisRun, error) {
	var run domain.AnalysisRun
	err := r.col.FindOne(ctx, bson.M{
		"organizationId":  organizationID,
		"clientRequestId": clientRequestID,
	}).Decode(&run)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *AnalysisRunRepository) Update(ctx context.Context, organizationID primitive.ObjectID, run *domain.AnalysisRun) error {
	run.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id":            run.ID,
		"organizationId": organizationID,
	}, bson.M{"$set": run})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AnalysisRunRepository) ListByOrg(ctx context.Context, organizationID primitive.ObjectID, status *domain.AnalysisRunStatus, limit int) ([]domain.AnalysisRun, error) {
	if limit <= 0 {
		limit = 50
	}
	filter := bson.M{"organizationId": organizationID}
	if status != nil {
		filter["status"] = *status
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var runs []domain.AnalysisRun
	if err := cur.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *AnalysisRunRepository) ListByCreatedByUser(ctx context.Context, userID primitive.ObjectID, limit int) ([]domain.AnalysisRun, error) {
	if limit <= 0 {
		limit = 200
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, bson.M{"createdByUserId": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var runs []domain.AnalysisRun
	if err := cur.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *AnalysisRunRepository) AnonymizeUserActor(ctx context.Context, userID primitive.ObjectID) error {
	anon := domain.AnonymizedActorUserID
	_, err := r.col.UpdateMany(ctx, bson.M{"createdByUserId": userID}, bson.M{"$set": bson.M{
		"createdByUserId": anon,
		"updatedAt":       time.Now().UTC(),
	}})
	if err != nil {
		return err
	}
	_, err = r.col.UpdateMany(ctx, bson.M{"cancelledByUserId": userID}, bson.M{"$set": bson.M{
		"cancelledByUserId": anon,
		"updatedAt":         time.Now().UTC(),
	}})
	return err
}

func (r *AnalysisRunRepository) ListStaleRunning(ctx context.Context, limit int) ([]domain.AnalysisRun, error) {
	if limit <= 0 {
		limit = 50
	}
	cutoff := time.Now().UTC().Add(-3 * time.Minute)
	filter := bson.M{
		"status": domain.AnalysisStatusRunning,
		"$or": []bson.M{
			{"lastHeartbeatAt": bson.M{"$lt": cutoff}},
			{"lastHeartbeatAt": bson.M{"$exists": false}, "startedAt": bson.M{"$lt": cutoff}},
		},
	}
	opts := options.Find().SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var runs []domain.AnalysisRun
	if err := cur.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}
