package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LLMProviderStatus string

const (
	LLMProviderActive   LLMProviderStatus = "ACTIVE"
	LLMProviderDisabled LLMProviderStatus = "DISABLED"
)

type LLMModelStatus string

const (
	LLMModelDraft     LLMModelStatus = "DRAFT"
	LLMModelActive    LLMModelStatus = "ACTIVE"
	LLMModelDegraded  LLMModelStatus = "DEGRADED"
	LLMModelDisabled  LLMModelStatus = "DISABLED"
	LLMModelRetired   LLMModelStatus = "RETIRED"
)

type LLMHealthStatus string

const (
	LLMHealthHealthy   LLMHealthStatus = "HEALTHY"
	LLMHealthDegraded  LLMHealthStatus = "DEGRADED"
	LLMHealthUnhealthy LLMHealthStatus = "UNHEALTHY"
	LLMHealthUnknown   LLMHealthStatus = "UNKNOWN"
)

type LLMConfigStatus string

const (
	LLMConfigDraft     LLMConfigStatus = "DRAFT"
	LLMConfigValidated LLMConfigStatus = "VALIDATED"
	LLMConfigPublished LLMConfigStatus = "PUBLISHED"
	LLMConfigRetired   LLMConfigStatus = "RETIRED"
)

const (
	ConfigSnapshotKindPhase6LLM = "PHASE6_LLM"

	PersonaCarefulAnalyst = "careful_analyst"
	PersonaResultAnalyst  = "result_analyst"

	LLMPlatformDefaultRoutingVersion = "1.0.0"
	LLMPlatformDefaultPersonaVersion = "1.0.0"
)

type LLMProvider struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ProviderKey string           `bson:"providerKey"`
	DisplayName string           `bson:"displayName"`
	Status      LLMProviderStatus `bson:"status"`
	SecretRef   string           `bson:"secretRef"`
	BaseURLRef  string           `bson:"baseUrlRef,omitempty"`
	CreatedAt   time.Time        `bson:"createdAt"`
	UpdatedAt   time.Time        `bson:"updatedAt"`
}

type LLMModelCapabilities struct {
	Chat              bool     `bson:"chat"`
	JSON              bool     `bson:"json"`
	Tools             bool     `bson:"tools"`
	SupportedTaskTypes []string `bson:"supportedTaskTypes"`
}

type LLMFineTuneMetadata struct {
	BaseModelID      string `bson:"baseModelId,omitempty"`
	AdapterVersion   string `bson:"adapterVersion,omitempty"`
	DatasetVersionID string `bson:"datasetVersionId,omitempty"`
	TrainingRunID    string `bson:"trainingRunId,omitempty"`
	EvalID           string `bson:"evalId,omitempty"`
	ApprovalStatus   string `bson:"approvalStatus,omitempty"`
	DeploymentStatus string `bson:"deploymentStatus,omitempty"`
	RollbackTarget   string `bson:"rollbackTarget,omitempty"`
	ModelCardRef     string `bson:"modelCardRef,omitempty"`
	LicenseStatus    string `bson:"licenseStatus,omitempty"`
}

type LLMModel struct {
	ID                   primitive.ObjectID   `bson:"_id,omitempty"`
	ProviderID           primitive.ObjectID   `bson:"providerId"`
	ModelKey             string               `bson:"modelKey"`
	DisplayName          string               `bson:"displayName"`
	Status               LLMModelStatus       `bson:"status"`
	Capabilities         LLMModelCapabilities `bson:"capabilities"`
	ContextWindowTokens  int                  `bson:"contextWindowTokens"`
	DefaultForPlatform   bool                 `bson:"defaultForPlatform"`
	FallbackForPlatform  bool                 `bson:"fallbackForPlatform"`
	HealthStatus         LLMHealthStatus      `bson:"healthStatus"`
	LastHealthCheckAt    *time.Time           `bson:"lastHealthCheckAt,omitempty"`
	ProviderModelName    string               `bson:"providerModelName"`
	FineTune             *LLMFineTuneMetadata `bson:"fineTune,omitempty"`
	OrganizationAllowlist []primitive.ObjectID `bson:"organizationAllowlist,omitempty"`
	CreatedAt            time.Time            `bson:"createdAt"`
	UpdatedAt            time.Time            `bson:"updatedAt"`
}

type LLMRoutingRule struct {
	TaskType       string   `bson:"taskType,omitempty"`
	SafetyRiskMax  string   `bson:"safetyRiskMax,omitempty"`
	RequireEvidence bool    `bson:"requireEvidence,omitempty"`
	PreferredModels []string `bson:"preferredModels"`
}

type LLMRoutingPolicy struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID *primitive.ObjectID `bson:"organizationId,omitempty"`
	Version        string              `bson:"version"`
	Status         LLMConfigStatus     `bson:"status"`
	Rules          []LLMRoutingRule    `bson:"rules"`
	DefaultModelKey string             `bson:"defaultModelKey"`
	FallbackModelKey string           `bson:"fallbackModelKey"`
	Reason         string              `bson:"reason,omitempty"`
	CreatedBy      primitive.ObjectID  `bson:"createdBy"`
	CreatedAt      time.Time           `bson:"createdAt"`
	PublishedAt    *time.Time          `bson:"publishedAt,omitempty"`
}

type LLMPersona struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID  *primitive.ObjectID `bson:"organizationId,omitempty"`
	PersonaKey      string              `bson:"personaKey"`
	Version         string              `bson:"version"`
	Status          LLMConfigStatus     `bson:"status"`
	DisplayName     string              `bson:"displayName"`
	SystemInstruction string            `bson:"systemInstruction"`
	Reason          string              `bson:"reason,omitempty"`
	CreatedBy       primitive.ObjectID  `bson:"createdBy"`
	CreatedAt       time.Time           `bson:"createdAt"`
	PublishedAt     *time.Time          `bson:"publishedAt,omitempty"`
}

type LLMConfigurationDraft struct {
	ID                    primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID        *primitive.ObjectID `bson:"organizationId,omitempty"`
	Status                LLMConfigStatus     `bson:"status"`
	RoutingPolicyVersion  string              `bson:"routingPolicyVersion"`
	PersonaKey            string              `bson:"personaKey"`
	PersonaVersion        string              `bson:"personaVersion"`
	DefaultModelKey       string              `bson:"defaultModelKey"`
	FallbackModelKey      string              `bson:"fallbackModelKey"`
	ValidationErrors      []string            `bson:"validationErrors,omitempty"`
	ValidatedAt           *time.Time          `bson:"validatedAt,omitempty"`
	PublishedSnapshotID   *primitive.ObjectID `bson:"publishedSnapshotId,omitempty"`
	Reason                string              `bson:"reason,omitempty"`
	CreatedBy             primitive.ObjectID  `bson:"createdBy"`
	UpdatedBy             primitive.ObjectID  `bson:"updatedBy"`
	CreatedAt             time.Time           `bson:"createdAt"`
	UpdatedAt             time.Time           `bson:"updatedAt"`
}

type LLMOrgSettings struct {
	ID                     primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID         primitive.ObjectID  `bson:"organizationId"`
	PersonaKey             string              `bson:"personaKey,omitempty"`
	DefaultModelKey        string              `bson:"defaultModelKey,omitempty"`
	FallbackModelKey       string              `bson:"fallbackModelKey,omitempty"`
	ActiveConfigSnapshotID *primitive.ObjectID `bson:"activeConfigSnapshotId,omitempty"`
	UpdatedBy              primitive.ObjectID  `bson:"updatedBy"`
	UpdatedAt              time.Time           `bson:"updatedAt"`
}

type LLMCallStatus string

const (
	LLMCallSucceeded LLMCallStatus = "SUCCEEDED"
	LLMCallFailed    LLMCallStatus = "FAILED"
	LLMCallBlocked   LLMCallStatus = "BLOCKED"
)

type LLMCall struct {
	ID                    primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID        primitive.ObjectID  `bson:"organizationId"`
	UserID                *primitive.ObjectID `bson:"userId,omitempty"`
	AnalysisRunID         *primitive.ObjectID `bson:"analysisRunId,omitempty"`
	CorrelationID         string              `bson:"correlationId"`
	IdempotencyKey        string              `bson:"idempotencyKey,omitempty"`
	ProviderKey           string              `bson:"providerKey"`
	ModelKey              string              `bson:"modelKey"`
	PersonaKey            string              `bson:"personaKey"`
	PersonaVersion        string              `bson:"personaVersion"`
	ConfigSnapshotID      *primitive.ObjectID `bson:"configSnapshotId,omitempty"`
	RoutingPolicyVersion  string              `bson:"routingPolicyVersion"`
	RoutingReason         string              `bson:"routingReason"`
	PrimaryModelKey       string              `bson:"primaryModelKey"`
	FallbackUsed          bool                `bson:"fallbackUsed"`
	RetryCount            int                 `bson:"retryCount"`
	InputTokens           int                 `bson:"inputTokens"`
	OutputTokens          int                 `bson:"outputTokens"`
	LatencyMS             int64               `bson:"latencyMs"`
	EstimatedCostUSD      float64             `bson:"estimatedCostUsd"`
	Status                LLMCallStatus       `bson:"status"`
	SafetyResult          string              `bson:"safetyResult,omitempty"`
	ComplianceProfile     string              `bson:"complianceProfile,omitempty"`
	CompliancePolicyVersion string            `bson:"compliancePolicyVersion,omitempty"`
	ComplianceReflexVersion string            `bson:"complianceReflexVersion,omitempty"`
	RedactionApplied      bool                `bson:"redactionApplied"`
	InputHash             string              `bson:"inputHash"`
	OutputHash            string              `bson:"outputHash,omitempty"`
	ErrorCode             string              `bson:"errorCode,omitempty"`
	CreatedAt             time.Time           `bson:"createdAt"`
}

type LLMUsageDaily struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID   primitive.ObjectID `bson:"organizationId"`
	Date             string             `bson:"date"`
	CallCount        int                `bson:"callCount"`
	InputTokens      int                `bson:"inputTokens"`
	OutputTokens     int                `bson:"outputTokens"`
	EstimatedCostUSD float64            `bson:"estimatedCostUsd"`
	FallbackCount    int                `bson:"fallbackCount"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
}

const (
	EventLLMConfigPublished   SecurityEventType = "LLM_CONFIG_PUBLISHED"
	EventLLMConfigRollback    SecurityEventType = "LLM_CONFIG_ROLLBACK"
	EventLLMPromptInjection   SecurityEventType = "LLM_PROMPT_INJECTION_BLOCKED"
	EventLLMCallBlocked       SecurityEventType = "LLM_CALL_BLOCKED"
)
