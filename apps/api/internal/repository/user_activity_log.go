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

type UserActivityLogRepository struct {
	col *mongo.Collection
}

func NewUserActivityLogRepository(db *mongo.Database) *UserActivityLogRepository {
	return &UserActivityLogRepository{col: db.Collection("user_activity_logs")}
}

func (r *UserActivityLogRepository) Create(ctx context.Context, entry *domain.UserActivityLog) error {
	entry.Timestamp = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, entry)
	if err != nil {
		return err
	}
	entry.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserActivityLogRepository) ListByUser(ctx context.Context, userID primitive.ObjectID, limit int, before *time.Time) ([]domain.UserActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	filter := bson.M{"userId": userID}
	if before != nil {
		filter["timestamp"] = bson.M{"$lt": *before}
	}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.UserActivityLog
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
