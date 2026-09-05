package mongo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultDBName = "miyuna"

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
	return &Client{DB: client.Database(databaseFromURI(uri))}, nil
}

func databaseFromURI(uri string) string {
	if u, err := url.Parse(uri); err == nil {
		if name := strings.TrimPrefix(u.Path, "/"); name != "" {
			return name
		}
	}
	return defaultDBName
}

func (c *Client) Ping(ctx context.Context) error {
	return c.DB.Client().Ping(ctx, nil)
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	ttl := options.Index().SetExpireAfterSeconds(0)
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
		{"sessions", mongo.IndexModel{Keys: bson.D{{Key: "refreshTokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"sessions", mongo.IndexModel{Keys: bson.D{{Key: "familyId", Value: 1}}}},
		{"sessions", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"rotated_refresh_tokens", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"rotated_refresh_tokens", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"pending_auth", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"pending_auth", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"email_verifications", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"email_verifications", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"device_verifications", mongo.IndexModel{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "deviceId", Value: 1}}}},
		{"device_verifications", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"mfa_setup_challenges", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"mfa_setup_challenges", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"organization_invitations", mongo.IndexModel{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"organization_invitations", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "email", Value: 1}, {Key: "status", Value: 1}}}},
		{"organization_invitations", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl}},
		{"compliance_policy_versions", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "profile", Value: 1}, {Key: "version", Value: -1}}}},
		{"compliance_policy_versions", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "status", Value: 1}, {Key: "effectiveAt", Value: -1}}}},
		{"consents", mongo.IndexModel{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "organizationId", Value: 1}, {Key: "purpose", Value: 1}, {Key: "withdrawnAt", Value: 1}}}},
		{"consents", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "purpose", Value: 1}}}},
		{"compliance_events", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "timestamp", Value: -1}}}},
		{"config_audit_log", mongo.IndexModel{Keys: bson.D{{Key: "changedAt", Value: -1}}}},
		{"config_audit_log", mongo.IndexModel{Keys: bson.D{{Key: "changedBy", Value: 1}, {Key: "changedAt", Value: -1}}}},
	}

	for _, idx := range indexes {
		if _, err := c.DB.Collection(idx.collection).Indexes().CreateOne(ctx, idx.model); err != nil {
			return fmt.Errorf("index %s: %w", idx.collection, err)
		}
	}

	phase4Indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{"products", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "createdAt", Value: -1}}}},
		{"product_source_mappings", mongo.IndexModel{
			Keys:    bson.D{{Key: "organizationId", Value: 1}, {Key: "source", Value: 1}, {Key: "sourceProductId", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"user_experiences", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "productId", Value: 1}}}},
		{"marketplace_reviews", mongo.IndexModel{
			Keys:    bson.D{{Key: "organizationId", Value: 1}, {Key: "fingerprint", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"marketplace_import_runs", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "createdAt", Value: -1}}}},
		{"raw_source_payloads", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "importRunId", Value: 1}}}},
		{"dataset_records", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "datasetEligibility", Value: 1}}}},
		{"dataset_records", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "datasetVersionId", Value: 1}}}},
		{"dataset_versions", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "createdAt", Value: -1}}}},
	}
	for _, idx := range phase4Indexes {
		if _, err := c.DB.Collection(idx.collection).Indexes().CreateOne(ctx, idx.model); err != nil {
			return fmt.Errorf("index %s: %w", idx.collection, err)
		}
	}

	phase5Indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{"analysis_runs", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "status", Value: 1}}}},
		{"analysis_runs", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "createdAt", Value: -1}}}},
		{"analysis_runs", mongo.IndexModel{
			Keys:    bson.D{{Key: "organizationId", Value: 1}, {Key: "clientRequestId", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"analysis_runs", mongo.IndexModel{Keys: bson.D{{Key: "status", Value: 1}, {Key: "lastHeartbeatAt", Value: 1}}}},
		{"agent_run_events", mongo.IndexModel{
			Keys:    bson.D{{Key: "organizationId", Value: 1}, {Key: "runId", Value: 1}, {Key: "sequence", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"tool_executions", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "analysisRunId", Value: 1}}}},
		{"config_snapshots", mongo.IndexModel{Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "publishedAt", Value: -1}}}},
	}
	for _, idx := range phase5Indexes {
		if _, err := c.DB.Collection(idx.collection).Indexes().CreateOne(ctx, idx.model); err != nil {
			return fmt.Errorf("index %s: %w", idx.collection, err)
		}
	}
	return nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.DB.Client().Disconnect(ctx)
}
