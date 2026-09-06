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

type LoginEmailVerificationRepository struct {
	col *mongo.Collection
}

func NewLoginEmailVerificationRepository(db *mongo.Database) *LoginEmailVerificationRepository {
	return &LoginEmailVerificationRepository{col: db.Collection("login_email_verifications")}
}

func (r *LoginEmailVerificationRepository) Create(ctx context.Context, lev *domain.LoginEmailVerification) error {
	lev.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, lev)
	if err != nil {
		return err
	}
	lev.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *LoginEmailVerificationRepository) FindActiveByPendingHash(ctx context.Context, pendingHash string) (*domain.LoginEmailVerification, error) {
	var lev domain.LoginEmailVerification
	err := r.col.FindOne(ctx, bson.M{
		"pendingHash": pendingHash,
		"locked":      false,
		"expiresAt":   bson.M{"$gt": time.Now().UTC()},
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&lev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &lev, nil
}

func (r *LoginEmailVerificationRepository) ConsumeByPendingHashAndCode(ctx context.Context, pendingHash, codeHash string) (*domain.LoginEmailVerification, error) {
	var lev domain.LoginEmailVerification
	err := r.col.FindOneAndDelete(ctx, bson.M{
		"pendingHash": pendingHash,
		"codeHash":    codeHash,
		"locked":      false,
		"expiresAt":   bson.M{"$gt": time.Now().UTC()},
	}).Decode(&lev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &lev, nil
}

func (r *LoginEmailVerificationRepository) IncrementFailedAttempts(ctx context.Context, id primitive.ObjectID, maxAttempts int) (locked bool, err error) {
	var lev domain.LoginEmailVerification
	if err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"failedAttempts": 1}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&lev); err != nil {
		return false, err
	}
	if lev.FailedAttempts >= maxAttempts {
		_, err = r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"locked": true}})
		return true, err
	}
	return false, nil
}

func (r *LoginEmailVerificationRepository) InvalidateByPendingHash(ctx context.Context, pendingHash string) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"pendingHash": pendingHash})
	return err
}
