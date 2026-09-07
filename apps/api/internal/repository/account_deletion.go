package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccountDeletionRepository struct {
	col *mongo.Collection
}

func NewAccountDeletionRepository(db *mongo.Database) *AccountDeletionRepository {
	return &AccountDeletionRepository{col: db.Collection("account_deletion_requests")}
}

func (r *AccountDeletionRepository) ReplacePending(ctx context.Context, req *domain.AccountDeletionRequest) error {
	_, _ = r.col.DeleteMany(ctx, bson.M{"userId": req.UserID})
	req.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, req)
	if err != nil {
		return err
	}
	req.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AccountDeletionRepository) FindByCodeHash(ctx context.Context, codeHash string) (*domain.AccountDeletionRequest, error) {
	var req domain.AccountDeletionRequest
	err := r.col.FindOne(ctx, bson.M{
		"codeHash":  codeHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&req)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *AccountDeletionRepository) ConsumeByCodeHash(ctx context.Context, userID primitive.ObjectID, codeHash string) (*domain.AccountDeletionRequest, error) {
	var req domain.AccountDeletionRequest
	err := r.col.FindOneAndDelete(ctx, bson.M{
		"userId":    userID,
		"codeHash":  codeHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&req)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *AccountDeletionRepository) DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"userId": userID})
	return err
}
