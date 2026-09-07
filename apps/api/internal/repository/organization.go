package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrganizationRepository struct {
	col *mongo.Collection
}

func NewOrganizationRepository(db *mongo.Database) *OrganizationRepository {
	return &OrganizationRepository{col: db.Collection("organizations")}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	now := time.Now().UTC()
	org.CreatedAt = now
	org.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, org)
	if err != nil {
		return err
	}
	org.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *OrganizationRepository) FindByID(ctx context.Context, organizationID primitive.ObjectID) (*domain.Organization, error) {
	var org domain.Organization
	err := r.col.FindOne(ctx, bson.M{"_id": organizationID}).Decode(&org)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepository) UpdateCompliance(ctx context.Context, orgID primitive.ObjectID, profile, policyVersion string) error {
	_, err := r.col.UpdateByID(ctx, orgID, bson.M{"$set": bson.M{
		"complianceProfile":       profile,
		"compliancePolicyVersion": policyVersion,
		"updatedAt":               time.Now().UTC(),
	}})
	return err
}

func (r *OrganizationRepository) Delete(ctx context.Context, orgID primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": orgID})
	return err
}
