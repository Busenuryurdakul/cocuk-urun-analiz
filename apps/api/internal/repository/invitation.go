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

type InvitationRepository struct {
	col *mongo.Collection
}

func NewInvitationRepository(db *mongo.Database) *InvitationRepository {
	return &InvitationRepository{col: db.Collection("organization_invitations")}
}

func (r *InvitationRepository) Create(ctx context.Context, inv *domain.OrganizationInvitation) error {
	now := time.Now().UTC()
	inv.CreatedAt = now
	res, err := r.col.InsertOne(ctx, inv)
	if err != nil {
		return err
	}
	inv.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *InvitationRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	err := r.col.FindOne(ctx, bson.M{"tokenHash": tokenHash}).Decode(&inv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) MarkAccepted(ctx context.Context, id, userID primitive.ObjectID) error {
	now := time.Now().UTC()
	_, err := r.col.UpdateOne(ctx, bson.M{
		"_id":    id,
		"status": domain.InvitationPending,
	}, bson.M{"$set": bson.M{
		"status":           domain.InvitationAccepted,
		"acceptedAt":       now,
		"acceptedByUserId": userID,
	}})
	return err
}

func (r *InvitationRepository) MarkExpired(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status": domain.InvitationExpired,
	}})
	return err
}

func (r *InvitationRepository) FindPendingByOrgAndEmail(ctx context.Context, orgID primitive.ObjectID, email string) (*domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	err := r.col.FindOne(ctx, bson.M{
		"organizationId": orgID,
		"email":          email,
		"status":         domain.InvitationPending,
	}).Decode(&inv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) ListPendingByOrg(ctx context.Context, orgID primitive.ObjectID) ([]domain.OrganizationInvitation, error) {
	cur, err := r.col.Find(ctx, bson.M{
		"organizationId": orgID,
		"status":         domain.InvitationPending,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.OrganizationInvitation
	return out, cur.All(ctx, &out)
}

func (r *InvitationRepository) ConsumeByTokenHash(ctx context.Context, tokenHash string) (*domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	err := r.col.FindOneAndUpdate(ctx,
		bson.M{"tokenHash": tokenHash, "status": domain.InvitationPending},
		bson.M{"$set": bson.M{"status": domain.InvitationAccepted, "acceptedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&inv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inv, nil
}
