package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DatasetVersionRepository struct {
	col *mongo.Collection
}

func NewDatasetVersionRepository(db *mongo.Database) *DatasetVersionRepository {
	return &DatasetVersionRepository{col: db.Collection("dataset_versions")}
}

func (r *DatasetVersionRepository) Create(ctx context.Context, version *domain.DatasetVersion) error {
	version.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, version)
	if err != nil {
		return err
	}
	version.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *DatasetVersionRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.DatasetVersion, error) {
	var version domain.DatasetVersion
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": organizationID}).Decode(&version)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &version, nil
}

func (r *DatasetVersionRepository) ListByOrg(ctx context.Context, organizationID primitive.ObjectID, limit int) ([]domain.DatasetVersion, error) {
	if limit <= 0 {
		limit = 20
	}
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID}, mongoOptionsLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var versions []domain.DatasetVersion
	if err := cur.All(ctx, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}
