package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepository struct {
	col *mongo.Collection
}

func NewSessionRepository(db *mongo.Database) *SessionRepository {
	return &SessionRepository{col: db.Collection("sessions")}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	now := time.Now().UTC()
	session.CreatedAt = now
	session.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, session)
	if err != nil {
		return err
	}
	session.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Session, error) {
	var session domain.Session
	err := r.col.FindOne(ctx, bson.M{
		"_id":       id,
		"revoked":   false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) FindByRefreshHash(ctx context.Context, refreshHash string) (*domain.Session, error) {
	var session domain.Session
	err := r.col.FindOne(ctx, bson.M{
		"refreshTokenHash": refreshHash,
		"revoked":          false,
		"expiresAt":        bson.M{"$gt": time.Now().UTC()},
	}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) RotateRefresh(ctx context.Context, sessionID primitive.ObjectID, newHash string, expiresAt time.Time) error {
	_, err := r.col.UpdateByID(ctx, sessionID, bson.M{"$set": bson.M{
		"refreshTokenHash": newHash,
		"expiresAt":        expiresAt,
		"updatedAt":        time.Now().UTC(),
	}})
	return err
}

func (r *SessionRepository) UpdateOrganization(ctx context.Context, sessionID, organizationID primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, sessionID, bson.M{"$set": bson.M{
		"organizationId": organizationID,
		"updatedAt":      time.Now().UTC(),
	}})
	return err
}

func (r *SessionRepository) RevokeByID(ctx context.Context, sessionID primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, sessionID, bson.M{"$set": bson.M{
		"revoked":   true,
		"updatedAt": time.Now().UTC(),
	}})
	return err
}

func (r *SessionRepository) RevokeFamily(ctx context.Context, familyID primitive.ObjectID) error {
	_, err := r.col.UpdateMany(ctx, bson.M{"familyId": familyID}, bson.M{"$set": bson.M{
		"revoked":   true,
		"updatedAt": time.Now().UTC(),
	}})
	return err
}

type RotatedRefreshRepository struct {
	col *mongo.Collection
}

func NewRotatedRefreshRepository(db *mongo.Database) *RotatedRefreshRepository {
	return &RotatedRefreshRepository{col: db.Collection("rotated_refresh_tokens")}
}

func (r *RotatedRefreshRepository) Record(ctx context.Context, rotated *domain.RotatedRefreshToken) error {
	rotated.RotatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, rotated)
	if err != nil {
		return err
	}
	rotated.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RotatedRefreshRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RotatedRefreshToken, error) {
	var rotated domain.RotatedRefreshToken
	err := r.col.FindOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&rotated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rotated, nil
}

type PendingAuthRepository struct {
	col *mongo.Collection
}

func NewPendingAuthRepository(db *mongo.Database) *PendingAuthRepository {
	return &PendingAuthRepository{col: db.Collection("pending_auth")}
}

func (r *PendingAuthRepository) Create(ctx context.Context, pending *domain.PendingAuth) error {
	pending.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, pending)
	if err != nil {
		return err
	}
	pending.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *PendingAuthRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PendingAuth, error) {
	var pending domain.PendingAuth
	err := r.col.FindOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&pending)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pending, nil
}

func (r *PendingAuthRepository) IncrementFailedAttempts(ctx context.Context, id primitive.ObjectID, maxAttempts int) (locked bool, err error) {
	var pending domain.PendingAuth
	err = r.col.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "locked": false},
		bson.M{"$inc": bson.M{"failedAttempts": 1}},
	).Decode(&pending)
	if err != nil {
		return false, err
	}
	attempts := pending.FailedAttempts + 1
	if attempts >= maxAttempts {
		_, _ = r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"locked": true}})
		return true, nil
	}
	return false, nil
}

func (r *PendingAuthRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"tokenHash": tokenHash})
	return err
}

type EmailVerificationRepository struct {
	col *mongo.Collection
}

func NewEmailVerificationRepository(db *mongo.Database) *EmailVerificationRepository {
	return &EmailVerificationRepository{col: db.Collection("email_verifications")}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, ev *domain.EmailVerification) error {
	ev.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, ev)
	if err != nil {
		return err
	}
	ev.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *EmailVerificationRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error) {
	var ev domain.EmailVerification
	err := r.col.FindOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&ev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ev, nil
}

func (r *EmailVerificationRepository) ConsumeByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error) {
	var ev domain.EmailVerification
	err := r.col.FindOneAndDelete(ctx, bson.M{
		"tokenHash": tokenHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&ev)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ev, nil
}

func (r *EmailVerificationRepository) LockByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"tokenHash": tokenHash}, bson.M{"$set": bson.M{"locked": true}})
	return err
}

type DeviceVerificationRepository struct {
	col *mongo.Collection
}

func NewDeviceVerificationRepository(db *mongo.Database) *DeviceVerificationRepository {
	return &DeviceVerificationRepository{col: db.Collection("device_verifications")}
}

func (r *DeviceVerificationRepository) Create(ctx context.Context, dv *domain.DeviceVerification) error {
	dv.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, dv)
	if err != nil {
		return err
	}
	dv.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *DeviceVerificationRepository) FindActive(ctx context.Context, userID, deviceID primitive.ObjectID) (*domain.DeviceVerification, error) {
	var dv domain.DeviceVerification
	err := r.col.FindOne(ctx, bson.M{
		"userId":    userID,
		"deviceId":  deviceID,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&dv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &dv, nil
}

func (r *DeviceVerificationRepository) ConsumeByUserDeviceAndCode(ctx context.Context, userID, deviceID primitive.ObjectID, codeHash string) (*domain.DeviceVerification, error) {
	var dv domain.DeviceVerification
	err := r.col.FindOneAndDelete(ctx, bson.M{
		"userId":    userID,
		"deviceId":  deviceID,
		"codeHash":  codeHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&dv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &dv, nil
}

func (r *DeviceVerificationRepository) IncrementFailedAttempts(ctx context.Context, id primitive.ObjectID, maxAttempts int) (locked bool, err error) {
	var dv domain.DeviceVerification
	err = r.col.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "locked": false},
		bson.M{"$inc": bson.M{"failedAttempts": 1}},
	).Decode(&dv)
	if err != nil {
		return false, err
	}
	if dv.FailedAttempts+1 >= maxAttempts {
		_, _ = r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"locked": true}})
		return true, nil
	}
	return false, nil
}

type MFASetupRepository struct {
	col *mongo.Collection
}

func NewMFASetupRepository(db *mongo.Database) *MFASetupRepository {
	return &MFASetupRepository{col: db.Collection("mfa_setup_challenges")}
}

func (r *MFASetupRepository) Create(ctx context.Context, challenge *domain.MFASetupChallenge) error {
	challenge.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, challenge)
	if err != nil {
		return err
	}
	challenge.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MFASetupRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.MFASetupChallenge, error) {
	var challenge domain.MFASetupChallenge
	err := r.col.FindOne(ctx, bson.M{
		"tokenHash": tokenHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&challenge)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &challenge, nil
}

func (r *MFASetupRepository) ConsumeByTokenHash(ctx context.Context, tokenHash string) (*domain.MFASetupChallenge, error) {
	var challenge domain.MFASetupChallenge
	err := r.col.FindOneAndDelete(ctx, bson.M{
		"tokenHash": tokenHash,
		"locked":    false,
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&challenge)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &challenge, nil
}

func (r *MFASetupRepository) IncrementFailedAttempts(ctx context.Context, id primitive.ObjectID, maxAttempts int) (locked bool, err error) {
	var challenge domain.MFASetupChallenge
	err = r.col.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "locked": false},
		bson.M{"$inc": bson.M{"failedAttempts": 1}},
	).Decode(&challenge)
	if err != nil {
		return false, err
	}
	if challenge.FailedAttempts+1 >= maxAttempts {
		_, _ = r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"locked": true}})
		return true, nil
	}
	return false, nil
}
