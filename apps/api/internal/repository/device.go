package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	_, err = r.col.UpdateByID(ctx, device.ID, bson.M{"$set": bson.M{"lastActiveAt": now}})
	return err
}

func (r *DeviceRepository) MarkVerified(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"verified":     true,
		"lastActiveAt": time.Now().UTC(),
	}})
	return err
}

func (r *DeviceRepository) FindByUserAndFingerprint(ctx context.Context, userID primitive.ObjectID, fingerprint string) (*domain.Device, error) {
	var device domain.Device
	err := r.col.FindOne(ctx, bson.M{
		"userId":            userID,
		"deviceFingerprint": fingerprint,
	}).Decode(&device)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &device, nil
}
