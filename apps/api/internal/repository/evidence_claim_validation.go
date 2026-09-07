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

type EvidenceClaimValidationRepository struct {
	col *mongo.Collection
}

func NewEvidenceClaimValidationRepository(db *mongo.Database) *EvidenceClaimValidationRepository {
	return &EvidenceClaimValidationRepository{col: db.Collection("evidence_claim_validations")}
}

func (r *EvidenceClaimValidationRepository) Upsert(ctx context.Context, v *domain.EvidenceClaimValidation) error {
	now := time.Now().UTC()
	if v.ID.IsZero() {
		v.ID = primitive.NewObjectID()
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	if v.EvidenceIDs == nil {
		v.EvidenceIDs = []primitive.ObjectID{}
	}
	if v.Issues == nil {
		v.Issues = []string{}
	}

	filter := bson.M{
		"organizationId": v.OrganizationID,
		"analysisRunId":  v.AnalysisRunID,
		"claimId":        v.ClaimID,
	}
	update := bson.M{
		"$set": bson.M{
			"claimText":     v.ClaimText,
			"evidenceIds":   v.EvidenceIDs,
			"supportStatus": v.SupportStatus,
			"issues":        v.Issues,
			"updatedAt":     v.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"_id":            v.ID,
			"organizationId": v.OrganizationID,
			"analysisRunId":  v.AnalysisRunID,
			"claimId":        v.ClaimID,
			"createdAt":      v.CreatedAt,
		},
	}
	opts := options.Update().SetUpsert(true)
	res, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.UpsertedID != nil {
		if id, ok := res.UpsertedID.(primitive.ObjectID); ok {
			v.ID = id
		}
	} else {
		var existing domain.EvidenceClaimValidation
		if err := r.col.FindOne(ctx, filter).Decode(&existing); err == nil {
			v.ID = existing.ID
			v.CreatedAt = existing.CreatedAt
		}
	}
	return nil
}

func (r *EvidenceClaimValidationRepository) FindByClaim(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, claimID string) (*domain.EvidenceClaimValidation, error) {
	var v domain.EvidenceClaimValidation
	err := r.col.FindOne(ctx, bson.M{
		"organizationId": organizationID,
		"analysisRunId":  analysisRunID,
		"claimId":        claimID,
	}).Decode(&v)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

func (r *EvidenceClaimValidationRepository) ListByAnalysisRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID, limit int) ([]domain.EvidenceClaimValidation, error) {
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, bson.M{
		"organizationId": organizationID,
		"analysisRunId":  analysisRunID,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var items []domain.EvidenceClaimValidation
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.EvidenceClaimValidation{}
	}
	return items, nil
}
