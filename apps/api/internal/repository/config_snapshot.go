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

type ConfigSnapshotRepository struct {
	col *mongo.Collection
}

func NewConfigSnapshotRepository(db *mongo.Database) *ConfigSnapshotRepository {
	return &ConfigSnapshotRepository{col: db.Collection("config_snapshots")}
}

func (r *ConfigSnapshotRepository) Create(ctx context.Context, snap *domain.ConfigSnapshot) error {
	snap.PublishedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, snap)
	if err != nil {
		return err
	}
	snap.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ConfigSnapshotRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.ConfigSnapshot, error) {
	var snap domain.ConfigSnapshot
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&snap)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &snap, nil
}

func (r *ConfigSnapshotRepository) FindLatestPublished(ctx context.Context, orgID *primitive.ObjectID, kind string) (*domain.ConfigSnapshot, error) {
	filter := bson.M{"snapshotKind": kind}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "publishedAt", Value: -1}})
	var snap domain.ConfigSnapshot
	err := r.col.FindOne(ctx, filter, opts).Decode(&snap)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &snap, nil
}

func (r *ConfigSnapshotRepository) ListPublished(ctx context.Context, orgID *primitive.ObjectID, kind string, limit int64) ([]domain.ConfigSnapshot, error) {
	if limit <= 0 {
		limit = 10
	}
	filter := bson.M{"snapshotKind": kind}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "publishedAt", Value: -1}}).SetLimit(limit)
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.ConfigSnapshot
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
