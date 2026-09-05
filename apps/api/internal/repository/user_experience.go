package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserExperienceRepository struct {
	col *mongo.Collection
}

func NewUserExperienceRepository(db *mongo.Database) *UserExperienceRepository {
	return &UserExperienceRepository{col: db.Collection("user_experiences")}
}

func (r *UserExperienceRepository) Create(ctx context.Context, ux *domain.UserExperience) error {
	now := time.Now().UTC()
	ux.CreatedAt = now
	ux.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, ux)
	if err != nil {
		return err
	}
	ux.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserExperienceRepository) FindByID(ctx context.Context, organizationID, id primitive.ObjectID) (*domain.UserExperience, error) {
	var ux domain.UserExperience
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": organizationID}).Decode(&ux)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ux, nil
}

func (r *UserExperienceRepository) Update(ctx context.Context, organizationID primitive.ObjectID, ux *domain.UserExperience) error {
	ux.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": ux.ID, "organizationId": organizationID}, bson.M{"$set": ux})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserExperienceRepository) Delete(ctx context.Context, organizationID, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "organizationId": organizationID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserExperienceRepository) ListByProduct(ctx context.Context, organizationID, productID primitive.ObjectID) ([]domain.UserExperience, error) {
	cur, err := r.col.Find(ctx, bson.M{"organizationId": organizationID, "productId": productID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var items []domain.UserExperience
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}
