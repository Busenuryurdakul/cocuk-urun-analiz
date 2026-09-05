package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
