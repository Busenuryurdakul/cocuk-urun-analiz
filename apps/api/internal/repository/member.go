package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MemberRepository struct {
	col *mongo.Collection
}

func NewMemberRepository(db *mongo.Database) *MemberRepository {
	return &MemberRepository{col: db.Collection("organization_members")}
}

func (r *MemberRepository) Create(ctx context.Context, member *domain.OrganizationMember) error {
	now := time.Now().UTC()
	member.InvitedAt = now
	member.JoinedAt = now
	res, err := r.col.InsertOne(ctx, member)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	member.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MemberRepository) FindByUserAndOrg(ctx context.Context, userID, organizationID primitive.ObjectID) (*domain.OrganizationMember, error) {
	var member domain.OrganizationMember
	err := r.col.FindOne(ctx, bson.M{
		"userId":         userID,
		"organizationId": organizationID,
	}).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *MemberRepository) ListByUser(ctx context.Context, userID primitive.ObjectID) ([]domain.OrganizationMember, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var members []domain.OrganizationMember
	if err := cur.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}
