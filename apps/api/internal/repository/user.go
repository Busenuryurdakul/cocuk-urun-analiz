package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	user.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := r.col.UpdateByID(ctx, user.ID, bson.M{"$set": user})
	return err
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"emailVerified": true,
		"updatedAt":     time.Now().UTC(),
	}})
	return err
}

func (r *UserRepository) EnableMFA(ctx context.Context, id primitive.ObjectID, secret string) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"mfaEnabled": true,
		"mfaSecret":  secret,
		"updatedAt":  time.Now().UTC(),
	}})
	return err
}
