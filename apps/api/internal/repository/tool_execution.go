package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ToolExecutionRepository struct {
	col *mongo.Collection
}

func NewToolExecutionRepository(db *mongo.Database) *ToolExecutionRepository {
	return &ToolExecutionRepository{col: db.Collection("tool_executions")}
}

func (r *ToolExecutionRepository) Create(ctx context.Context, exec *domain.ToolExecution) error {
	exec.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, exec)
	if err != nil {
		return err
	}
	exec.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ToolExecutionRepository) Update(ctx context.Context, organizationID primitive.ObjectID, exec *domain.ToolExecution) error {
	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id":            exec.ID,
		"organizationId": organizationID,
	}, bson.M{"$set": exec})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ToolExecutionRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.ToolExecution, error) {
	var exec domain.ToolExecution
	err := r.col.FindOne(ctx, bson.M{
		"_id":            id,
		"organizationId": organizationID,
	}).Decode(&exec)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &exec, nil
}
