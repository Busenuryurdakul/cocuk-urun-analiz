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

type DeviceRepository struct {
	col *mongo.Collection
}

func NewDeviceRepository(db *mongo.Database) *DeviceRepository {
	return &DeviceRepository{col: db.Collection("devices")}
}

func (r *DeviceRepository) Upsert(ctx context.Context, device *domain.Device) error {
	now := time.Now().UTC()
	filter := bson.M{
		"userId":            device.UserID,
		"deviceFingerprint": device.DeviceFingerprint,
	}
	var existing domain.Device
	err := r.col.FindOne(ctx, filter).Decode(&existing)
	if err == mongo.ErrNoDocuments {
		device.CreatedAt = now
		device.LastActiveAt = now
		res, insErr := r.col.InsertOne(ctx, device)
		if insErr != nil {
			return insErr
		}
		device.ID = res.InsertedID.(primitive.ObjectID)
		return nil
	}
	if err != nil {
		return err
	}
	device.ID = existing.ID
	device.Verified = existing.Verified
	device.CreatedAt = existing.CreatedAt
	device.LastActiveAt = now
	if existing.RevokedAt != nil {
		device.RevokedAt = nil
	}
	set := bson.M{
		"lastActiveAt": now,
		"platform":     device.Platform,
	}
	if device.Label != "" {
		set["label"] = device.Label
	}
	if device.UserAgent != "" {
		set["userAgent"] = device.UserAgent
	}
	if device.IPAddress != "" {
		set["ipAddress"] = device.IPAddress
	}
	if device.AppVersion != "" {
		set["appVersion"] = device.AppVersion
	}
	if device.RevokedAt == nil && existing.RevokedAt != nil {
		set["revokedAt"] = nil
	}
	_, err = r.col.UpdateByID(ctx, device.ID, bson.M{"$set": set})
	return err
}

func (r *DeviceRepository) MarkVerified(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"verified":     true,
		"lastActiveAt": time.Now().UTC(),
	}})
	return err
}

func (r *DeviceRepository) TouchLastActive(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"lastActiveAt": time.Now().UTC()}})
	return err
}

func (r *DeviceRepository) FindByUserAndFingerprint(ctx context.Context, userID primitive.ObjectID, fingerprint string) (*domain.Device, error) {
	var device domain.Device
	err := r.col.FindOne(ctx, bson.M{
		"userId":            userID,
		"deviceFingerprint": fingerprint,
		"revokedAt":         nil,
	}).Decode(&device)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) FindByID(ctx context.Context, userID, deviceID primitive.ObjectID) (*domain.Device, error) {
	var device domain.Device
	err := r.col.FindOne(ctx, bson.M{
		"_id":       deviceID,
		"userId":    userID,
		"revokedAt": nil,
	}).Decode(&device)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) ListByUser(ctx context.Context, userID primitive.ObjectID) ([]domain.Device, error) {
	cur, err := r.col.Find(ctx, bson.M{
		"userId":    userID,
		"revokedAt": nil,
	}, options.Find().SetSort(bson.D{{Key: "lastActiveAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Device
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *DeviceRepository) Revoke(ctx context.Context, userID, deviceID primitive.ObjectID) error {
	now := time.Now().UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id":       deviceID,
		"userId":    userID,
		"revokedAt": nil,
	}, bson.M{"$set": bson.M{
		"revokedAt": now,
		"verified":  false,
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
