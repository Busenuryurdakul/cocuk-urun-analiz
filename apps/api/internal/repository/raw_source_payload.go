package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RawSourcePayloadRepository struct {
	col *mongo.Collection
}

func NewRawSourcePayloadRepository(db *mongo.Database) *RawSourcePayloadRepository {
	return &RawSourcePayloadRepository{col: db.Collection("raw_source_payloads")}
}

func (r *RawSourcePayloadRepository) Create(ctx context.Context, payload *domain.RawSourcePayload) error {
	payload.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, payload)
	if err != nil {
		return err
	}
	payload.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RawSourcePayloadRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.RawSourcePayload, error) {
	var payload domain.RawSourcePayload
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": organizationID}).Decode(&payload)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &payload, nil
}
