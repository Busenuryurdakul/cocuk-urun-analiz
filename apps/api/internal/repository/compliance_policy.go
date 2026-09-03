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

type CompliancePolicyRepository struct {
	col *mongo.Collection
}

func NewCompliancePolicyRepository(db *mongo.Database) *CompliancePolicyRepository {
	return &CompliancePolicyRepository{col: db.Collection("compliance_policy_versions")}
}

func (r *CompliancePolicyRepository) Insert(ctx context.Context, policy *domain.CompliancePolicyVersion) error {
	now := time.Now().UTC()
	policy.CreatedAt = now
	if policy.EffectiveAt.IsZero() {
		policy.EffectiveAt = now
	}
	res, err := r.col.InsertOne(ctx, policy)
	if err != nil {
		return err
	}
	policy.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *CompliancePolicyRepository) FindPublished(ctx context.Context, orgID *primitive.ObjectID, profile domain.ComplianceProfile, version string) (*domain.CompliancePolicyVersion, error) {
	filter := bson.M{
		"profile": profile,
		"version": version,
		"status":  domain.PolicyStatusPublished,
	}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	var policy domain.CompliancePolicyVersion
	err := r.col.FindOne(ctx, filter).Decode(&policy)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &policy, nil
}

func (r *CompliancePolicyRepository) FindLatestPublished(ctx context.Context, orgID *primitive.ObjectID, profile domain.ComplianceProfile) (*domain.CompliancePolicyVersion, error) {
	filter := bson.M{
		"profile": profile,
		"status":  domain.PolicyStatusPublished,
	}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "effectiveAt", Value: -1}})
	var policy domain.CompliancePolicyVersion
	err := r.col.FindOne(ctx, filter, opts).Decode(&policy)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &policy, nil
}

func (r *CompliancePolicyRepository) ListPublishedByOrg(ctx context.Context, orgID *primitive.ObjectID, limit int64) ([]domain.CompliancePolicyVersion, error) {
	filter := bson.M{"status": domain.PolicyStatusPublished}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "effectiveAt", Value: -1}}).SetLimit(limit)
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.CompliancePolicyVersion
	return out, cur.All(ctx, &out)
}

func (r *CompliancePolicyRepository) ArchiveOrgPublished(ctx context.Context, orgID primitive.ObjectID, profile domain.ComplianceProfile) error {
	_, err := r.col.UpdateMany(ctx, bson.M{
		"organizationId": orgID,
		"profile":        profile,
		"status":         domain.PolicyStatusPublished,
	}, bson.M{"$set": bson.M{"status": domain.PolicyStatusArchived}})
	return err
}

func (r *CompliancePolicyRepository) CountPlatformPublished(ctx context.Context, profile domain.ComplianceProfile, version string) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{
		"organizationId": nil,
		"profile":        profile,
		"version":        version,
		"status":         domain.PolicyStatusPublished,
	})
}

func (r *CompliancePolicyRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status domain.PolicyStatus) error {
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": id, "status": domain.PolicyStatusDraft}, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
