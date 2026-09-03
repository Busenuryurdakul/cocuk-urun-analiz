package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbName = "miyuna"

type Client struct {
	DB *mongo.Database
}

func Connect(ctx context.Context, uri string) (*Client, error) {
	opts := options.Client().ApplyURI(uri).SetConnectTimeout(10 * time.Second)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongodb ping: %w", err)
	}
	return &Client{DB: client.Database(dbName)}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.DB.Client().Ping(ctx, nil)
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{"users", mongo.IndexModel{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"organizations", mongo.IndexModel{Keys: bson.D{{Key: "ownerId", Value: 1}}}},
		{"organization_members", mongo.IndexModel{
			Keys:    bson.D{{Key: "organizationId", Value: 1}, {Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"organization_members", mongo.IndexModel{Keys: bson.D{{Key: "userId", Value: 1}}}},
		{"devices", mongo.IndexModel{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "deviceFingerprint", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"security_events", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "timestamp", Value: -1}}}},
		{"sessions", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"sessions", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		{"pending_auth", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"email_verifications", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"device_verifications", mongo.IndexModel{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "deviceId", Value: 1}}}},
		{"mfa_setup_challenges", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
	}

	for _, idx := range indexes {
		if _, err := c.DB.Collection(idx.collection).Indexes().CreateOne(ctx, idx.model); err != nil {
			return fmt.Errorf("index %s: %w", idx.collection, err)
		}
	}
	return nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.DB.Client().Disconnect(ctx)
}
