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

type ConsentRepository struct {
	col *mongo.Collection
}

func NewConsentRepository(db *mongo.Database) *ConsentRepository {
	return &ConsentRepository{col: db.Collection("consents")}
}

func (r *ConsentRepository) Insert(ctx context.Context, consent *domain.Consent) error {
	consent.GrantedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, consent)
	if err != nil {
		return err
	}
	consent.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ConsentRepository) FindActive(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose) (*domain.Consent, error) {
	filter := bson.M{
		"userId":      userID,
		"purpose":     purpose,
		"withdrawnAt": nil,
	}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "grantedAt", Value: -1}})
	var consent domain.Consent
	err := r.col.FindOne(ctx, filter, opts).Decode(&consent)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &consent, nil
}

func (r *ConsentRepository) Withdraw(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose) error {
	filter := bson.M{
		"userId":      userID,
		"purpose":     purpose,
		"withdrawnAt": nil,
	}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	now := time.Now().UTC()
	_, err := r.col.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"withdrawnAt": now}})
	return err
}

func (r *ConsentRepository) ListByUser(ctx context.Context, userID primitive.ObjectID) ([]domain.Consent, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": userID}, options.Find().SetSort(bson.D{{Key: "grantedAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Consent
	return out, cur.All(ctx, &out)
}

func (r *ConsentRepository) ListByOrg(ctx context.Context, orgID primitive.ObjectID) ([]domain.Consent, error) {
	cur, err := r.col.Find(ctx, bson.M{"organizationId": orgID}, options.Find().SetSort(bson.D{{Key: "grantedAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Consent
	return out, cur.All(ctx, &out)
}
