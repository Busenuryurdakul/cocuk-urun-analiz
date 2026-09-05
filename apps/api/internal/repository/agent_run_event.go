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

type AgentRunEventRepository struct {
	col *mongo.Collection
}

func NewAgentRunEventRepository(db *mongo.Database) *AgentRunEventRepository {
	return &AgentRunEventRepository{col: db.Collection("agent_run_events")}
}

func (r *AgentRunEventRepository) Append(ctx context.Context, event *domain.AgentRunEvent) error {
	event.Timestamp = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, event)
	if err != nil {
		return err
	}
	event.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AgentRunEventRepository) NextSequence(ctx context.Context, organizationID, runID primitive.ObjectID) (int64, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "sequence", Value: -1}})
	var last domain.AgentRunEvent
	err := r.col.FindOne(ctx, bson.M{
		"organizationId": organizationID,
		"runId":          runID,
	}, opts).Decode(&last)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 1, nil
		}
		return 0, err
	}
	return last.Sequence + 1, nil
}

func (r *AgentRunEventRepository) ListByRun(ctx context.Context, organizationID, runID primitive.ObjectID, afterSequence int64, limit int) ([]domain.AgentRunEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	filter := bson.M{
		"organizationId": organizationID,
		"runId":          runID,
	}
	if afterSequence > 0 {
		filter["sequence"] = bson.M{"$gt": afterSequence}
	}
	opts := options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var events []domain.AgentRunEvent
	if err := cur.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}
