package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SecurityEventRepository struct {
	col *mongo.Collection
}

func NewSecurityEventRepository(db *mongo.Database) *SecurityEventRepository {
	return &SecurityEventRepository{col: db.Collection("security_events")}
}

func (r *SecurityEventRepository) Record(ctx context.Context, event domain.SecurityEvent) error {
	event.Timestamp = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, event)
	return err
}

func (r *SecurityEventRepository) CountByType(ctx context.Context, eventType domain.SecurityEventType) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{"eventType": eventType})
}

func (r *SecurityEventRepository) ListByUser(ctx context.Context, userID primitive.ObjectID, limit int) ([]domain.SecurityEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	cur, err := r.col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.SecurityEvent
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
