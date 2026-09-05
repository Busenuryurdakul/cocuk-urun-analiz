package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DatasetRecordRepository struct {
	col *mongo.Collection
}

func NewDatasetRecordRepository(db *mongo.Database) *DatasetRecordRepository {
	return &DatasetRecordRepository{col: db.Collection("dataset_records")}
}

func (r *DatasetRecordRepository) Create(ctx context.Context, record *domain.DatasetRecord) error {
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, record)
	if err != nil {
		return err
	}
	record.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *DatasetRecordRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.DatasetRecord, error) {
	var record domain.DatasetRecord
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": organizationID}).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *DatasetRecordRepository) ListEligible(ctx context.Context, organizationID primitive.ObjectID, eligibility domain.DatasetEligibility) ([]domain.DatasetRecord, error) {
	filter := bson.M{
		"organizationId":     organizationID,
		"datasetEligibility": eligibility,
		"$or": []bson.M{
			{"split": bson.M{"$exists": false}},
			{"split": bson.M{"$ne": domain.SplitHeldOutEval}},
		},
	}
	cur, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var records []domain.DatasetRecord
	if err := cur.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *DatasetRecordRepository) ListByVersion(ctx context.Context, organizationID, versionID primitive.ObjectID) ([]domain.DatasetRecord, error) {
	cur, err := r.col.Find(ctx, bson.M{
		"organizationId":   organizationID,
		"datasetVersionId": versionID,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var records []domain.DatasetRecord
	if err := cur.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *DatasetRecordRepository) CountByEligibility(ctx context.Context, organizationID primitive.ObjectID) (map[string]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"organizationId": organizationID}}},
		{{Key: "$group", Value: bson.M{"_id": "$datasetEligibility", "count": bson.M{"$sum": 1}}}},
	}
	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[string]int{}
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		out[row.ID] = row.Count
	}
	return out, cur.Err()
}

func (r *DatasetRecordRepository) CountByRecordType(ctx context.Context, organizationID primitive.ObjectID) (map[string]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"organizationId": organizationID}}},
		{{Key: "$group", Value: bson.M{"_id": "$recordType", "count": bson.M{"$sum": 1}}}},
	}
	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[string]int{}
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		out[row.ID] = row.Count
	}
	return out, cur.Err()
}

func (r *DatasetRecordRepository) Update(ctx context.Context, organizationID primitive.ObjectID, record *domain.DatasetRecord) error {
	record.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": record.ID, "organizationId": organizationID}, bson.M{"$set": record})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
