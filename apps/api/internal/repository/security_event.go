package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
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
