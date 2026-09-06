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

type LLMProviderRepository struct {
	col *mongo.Collection
}

func NewLLMProviderRepository(db *mongo.Database) *LLMProviderRepository {
	return &LLMProviderRepository{col: db.Collection("llm_providers")}
}

func (r *LLMProviderRepository) Upsert(ctx context.Context, provider *domain.LLMProvider) error {
	now := time.Now().UTC()
	provider.UpdatedAt = now
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}
	filter := bson.M{"providerKey": provider.ProviderKey}
	update := bson.M{
		"$set": bson.M{
			"displayName": provider.DisplayName,
			"status":      provider.Status,
			"secretRef":   provider.SecretRef,
			"baseUrlRef":  provider.BaseURLRef,
			"updatedAt":   provider.UpdatedAt,
		},
		"$setOnInsert": bson.M{"createdAt": provider.CreatedAt, "providerKey": provider.ProviderKey},
	}
	opts := options.Update().SetUpsert(true)
	res, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.UpsertedID != nil {
		provider.ID = res.UpsertedID.(primitive.ObjectID)
	} else {
		existing, err := r.FindByKey(ctx, provider.ProviderKey)
		if err != nil {
			return err
		}
		provider.ID = existing.ID
	}
	return nil
}

func (r *LLMProviderRepository) FindByKey(ctx context.Context, key string) (*domain.LLMProvider, error) {
	var out domain.LLMProvider
	err := r.col.FindOne(ctx, bson.M{"providerKey": key}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMProviderRepository) List(ctx context.Context) ([]domain.LLMProvider, error) {
	cur, err := r.col.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "providerKey", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMProvider
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type LLMModelRepository struct {
	col *mongo.Collection
}

func NewLLMModelRepository(db *mongo.Database) *LLMModelRepository {
	return &LLMModelRepository{col: db.Collection("llm_models")}
}

func (r *LLMModelRepository) Upsert(ctx context.Context, model *domain.LLMModel) error {
	now := time.Now().UTC()
	model.UpdatedAt = now
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	filter := bson.M{"modelKey": model.ModelKey}
	update := bson.M{
		"$set": bson.M{
			"providerId":            model.ProviderID,
			"displayName":           model.DisplayName,
			"status":                model.Status,
			"capabilities":          model.Capabilities,
			"contextWindowTokens":   model.ContextWindowTokens,
			"defaultForPlatform":    model.DefaultForPlatform,
			"fallbackForPlatform":   model.FallbackForPlatform,
			"healthStatus":          model.HealthStatus,
			"providerModelName":     model.ProviderModelName,
			"fineTune":              model.FineTune,
			"organizationAllowlist": model.OrganizationAllowlist,
			"updatedAt":             model.UpdatedAt,
		},
		"$setOnInsert": bson.M{"createdAt": model.CreatedAt, "modelKey": model.ModelKey},
	}
	opts := options.Update().SetUpsert(true)
	res, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.UpsertedID != nil {
		model.ID = res.UpsertedID.(primitive.ObjectID)
	} else {
		existing, err := r.FindByKey(ctx, model.ModelKey)
		if err != nil {
			return err
		}
		model.ID = existing.ID
	}
	return nil
}

func (r *LLMModelRepository) FindByKey(ctx context.Context, key string) (*domain.LLMModel, error) {
	var out domain.LLMModel
	err := r.col.FindOne(ctx, bson.M{"modelKey": key}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMModelRepository) ListActive(ctx context.Context) ([]domain.LLMModel, error) {
	cur, err := r.col.Find(ctx, bson.M{
		"status": bson.M{"$in": []domain.LLMModelStatus{domain.LLMModelActive, domain.LLMModelDegraded}},
	}, options.Find().SetSort(bson.D{{Key: "modelKey", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMModel
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LLMModelRepository) ListAll(ctx context.Context) ([]domain.LLMModel, error) {
	cur, err := r.col.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "modelKey", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMModel
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LLMModelRepository) UpdateHealth(ctx context.Context, modelKey string, health domain.LLMHealthStatus) error {
	now := time.Now().UTC()
	_, err := r.col.UpdateOne(ctx, bson.M{"modelKey": modelKey}, bson.M{
		"$set": bson.M{
			"healthStatus":      health,
			"lastHealthCheckAt": now,
			"updatedAt":         now,
		},
	})
	return err
}

type LLMRoutingPolicyRepository struct {
	col *mongo.Collection
}

func NewLLMRoutingPolicyRepository(db *mongo.Database) *LLMRoutingPolicyRepository {
	return &LLMRoutingPolicyRepository{col: db.Collection("llm_routing_policies")}
}

func (r *LLMRoutingPolicyRepository) Insert(ctx context.Context, policy *domain.LLMRoutingPolicy) error {
	now := time.Now().UTC()
	policy.CreatedAt = now
	res, err := r.col.InsertOne(ctx, policy)
	if err != nil {
		return err
	}
	policy.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *LLMRoutingPolicyRepository) FindPublished(ctx context.Context, orgID *primitive.ObjectID, version string) (*domain.LLMRoutingPolicy, error) {
	filter := bson.M{"version": version, "status": domain.LLMConfigPublished}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	var out domain.LLMRoutingPolicy
	err := r.col.FindOne(ctx, filter).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMRoutingPolicyRepository) ArchivePublished(ctx context.Context, orgID *primitive.ObjectID) error {
	filter := bson.M{"status": domain.LLMConfigPublished}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	_, err := r.col.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"status": domain.LLMConfigRetired}})
	return err
}

func (r *LLMRoutingPolicyRepository) List(ctx context.Context, orgID *primitive.ObjectID, limit int64) ([]domain.LLMRoutingPolicy, error) {
	filter := bson.M{}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMRoutingPolicy
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type LLMPersonaRepository struct {
	col *mongo.Collection
}

func NewLLMPersonaRepository(db *mongo.Database) *LLMPersonaRepository {
	return &LLMPersonaRepository{col: db.Collection("llm_personas")}
}

func (r *LLMPersonaRepository) Insert(ctx context.Context, persona *domain.LLMPersona) error {
	now := time.Now().UTC()
	persona.CreatedAt = now
	res, err := r.col.InsertOne(ctx, persona)
	if err != nil {
		return err
	}
	persona.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *LLMPersonaRepository) FindPublished(ctx context.Context, orgID *primitive.ObjectID, personaKey, version string) (*domain.LLMPersona, error) {
	filter := bson.M{"personaKey": personaKey, "version": version, "status": domain.LLMConfigPublished}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	var out domain.LLMPersona
	err := r.col.FindOne(ctx, filter).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMPersonaRepository) List(ctx context.Context, orgID *primitive.ObjectID, limit int64) ([]domain.LLMPersona, error) {
	filter := bson.M{}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMPersona
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type LLMConfigurationDraftRepository struct {
	col *mongo.Collection
}

func NewLLMConfigurationDraftRepository(db *mongo.Database) *LLMConfigurationDraftRepository {
	return &LLMConfigurationDraftRepository{col: db.Collection("llm_configuration_drafts")}
}

func (r *LLMConfigurationDraftRepository) Insert(ctx context.Context, draft *domain.LLMConfigurationDraft) error {
	now := time.Now().UTC()
	draft.CreatedAt = now
	draft.UpdatedAt = now
	res, err := r.col.InsertOne(ctx, draft)
	if err != nil {
		return err
	}
	draft.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *LLMConfigurationDraftRepository) Update(ctx context.Context, draft *domain.LLMConfigurationDraft) error {
	draft.UpdatedAt = time.Now().UTC()
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": draft.ID}, draft)
	return err
}

func (r *LLMConfigurationDraftRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.LLMConfigurationDraft, error) {
	var out domain.LLMConfigurationDraft
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMConfigurationDraftRepository) List(ctx context.Context, orgID *primitive.ObjectID, limit int64) ([]domain.LLMConfigurationDraft, error) {
	filter := bson.M{}
	if orgID != nil {
		filter["organizationId"] = orgID
	} else {
		filter["organizationId"] = nil
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMConfigurationDraft
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type LLMOrgSettingsRepository struct {
	col *mongo.Collection
}

func NewLLMOrgSettingsRepository(db *mongo.Database) *LLMOrgSettingsRepository {
	return &LLMOrgSettingsRepository{col: db.Collection("llm_org_settings")}
}

func (r *LLMOrgSettingsRepository) Upsert(ctx context.Context, settings *domain.LLMOrgSettings) error {
	settings.UpdatedAt = time.Now().UTC()
	filter := bson.M{"organizationId": settings.OrganizationID}
	update := bson.M{"$set": settings}
	opts := options.Update().SetUpsert(true)
	res, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.UpsertedID != nil {
		settings.ID = res.UpsertedID.(primitive.ObjectID)
	}
	return nil
}

func (r *LLMOrgSettingsRepository) FindByOrg(ctx context.Context, orgID primitive.ObjectID) (*domain.LLMOrgSettings, error) {
	var out domain.LLMOrgSettings
	err := r.col.FindOne(ctx, bson.M{"organizationId": orgID}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

type LLMCallRepository struct {
	col *mongo.Collection
}

func NewLLMCallRepository(db *mongo.Database) *LLMCallRepository {
	return &LLMCallRepository{col: db.Collection("llm_calls")}
}

func (r *LLMCallRepository) Insert(ctx context.Context, call *domain.LLMCall) error {
	call.CreatedAt = time.Now().UTC()
	res, err := r.col.InsertOne(ctx, call)
	if err != nil {
		return err
	}
	call.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *LLMCallRepository) FindByID(ctx context.Context, orgID, id primitive.ObjectID) (*domain.LLMCall, error) {
	var out domain.LLMCall
	err := r.col.FindOne(ctx, bson.M{"_id": id, "organizationId": orgID}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *LLMCallRepository) FindByIdempotency(ctx context.Context, orgID primitive.ObjectID, key string) (*domain.LLMCall, error) {
	if key == "" {
		return nil, ErrNotFound
	}
	var out domain.LLMCall
	err := r.col.FindOne(ctx, bson.M{"organizationId": orgID, "idempotencyKey": key}).Decode(&out)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

type LLMUsageDailyRepository struct {
	col *mongo.Collection
}

func NewLLMUsageDailyRepository(db *mongo.Database) *LLMUsageDailyRepository {
	return &LLMUsageDailyRepository{col: db.Collection("llm_usage_daily")}
}

func (r *LLMUsageDailyRepository) Increment(ctx context.Context, orgID primitive.ObjectID, date string, inputTokens, outputTokens int, cost float64, fallback bool) error {
	inc := bson.M{
		"callCount":        1,
		"inputTokens":      inputTokens,
		"outputTokens":     outputTokens,
		"estimatedCostUsd": cost,
	}
	if fallback {
		inc["fallbackCount"] = 1
	}
	_, err := r.col.UpdateOne(ctx,
		bson.M{"organizationId": orgID, "date": date},
		bson.M{
			"$inc": inc,
			"$set": bson.M{"updatedAt": time.Now().UTC()},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *LLMUsageDailyRepository) SumSince(ctx context.Context, orgID primitive.ObjectID, fromDate string) (callCount, inputTokens, outputTokens int, cost float64, err error) {
	cur, err := r.col.Find(ctx, bson.M{"organizationId": orgID, "date": bson.M{"$gte": fromDate}})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row domain.LLMUsageDaily
		if err := cur.Decode(&row); err != nil {
			return 0, 0, 0, 0, err
		}
		callCount += row.CallCount
		inputTokens += row.InputTokens
		outputTokens += row.OutputTokens
		cost += row.EstimatedCostUSD
	}
	return callCount, inputTokens, outputTokens, cost, cur.Err()
}

func (r *LLMCallRepository) ListRecent(ctx context.Context, orgID primitive.ObjectID, limit int) ([]domain.LLMCall, error) {
	if limit <= 0 {
		limit = 20
	}
	cur, err := r.col.Find(
		ctx,
		bson.M{"organizationId": orgID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.LLMCall
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ModelUsageAggregate struct {
	ModelKey         string
	CallCount        int
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
}

func (r *LLMCallRepository) AggregateUsageByModel(ctx context.Context, orgID primitive.ObjectID, from time.Time) ([]ModelUsageAggregate, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"organizationId": orgID,
			"createdAt":      bson.M{"$gte": from},
			"status":         domain.LLMCallSucceeded,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":              "$modelKey",
			"callCount":        bson.M{"$sum": 1},
			"inputTokens":      bson.M{"$sum": "$inputTokens"},
			"outputTokens":     bson.M{"$sum": "$outputTokens"},
			"estimatedCostUsd": bson.M{"$sum": "$estimatedCostUsd"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "callCount", Value: -1}}}},
	}
	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []ModelUsageAggregate
	for cur.Next(ctx) {
		var row struct {
			ID               string  `bson:"_id"`
			CallCount        int     `bson:"callCount"`
			InputTokens      int     `bson:"inputTokens"`
			OutputTokens     int     `bson:"outputTokens"`
			EstimatedCostUSD float64 `bson:"estimatedCostUsd"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		out = append(out, ModelUsageAggregate{
			ModelKey:         row.ID,
			CallCount:        row.CallCount,
			InputTokens:      row.InputTokens,
			OutputTokens:     row.OutputTokens,
			EstimatedCostUSD: row.EstimatedCostUSD,
		})
	}
	return out, cur.Err()
}
