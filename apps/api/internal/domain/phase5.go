package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalysisRunStatus string

const (
	AnalysisStatusPending   AnalysisRunStatus = "PENDING"
	AnalysisStatusRunning   AnalysisRunStatus = "RUNNING"
	AnalysisStatusCompleted AnalysisRunStatus = "COMPLETED"
	AnalysisStatusFailed    AnalysisRunStatus = "FAILED"
	AnalysisStatusRejected  AnalysisRunStatus = "REJECTED"
)

type AgentEventStatus string

const (
	AgentEventRunning   AgentEventStatus = "RUNNING"
	AgentEventCompleted AgentEventStatus = "COMPLETED"
	AgentEventFailed    AgentEventStatus = "FAILED"
)

const (
	AgentPhaseRunStarted              = "RUN_STARTED"
	AgentPhaseCompliancePrecheck      = "COMPLIANCE_PRECHECK"
	AgentPhasePlanCreated             = "PLAN_CREATED"
	AgentPhaseToolSelected            = "TOOL_SELECTED"
	AgentPhaseToolExecutionStarted    = "TOOL_EXECUTION_STARTED"
	AgentPhaseToolExecutionCompleted  = "TOOL_EXECUTION_COMPLETED"
	AgentPhaseObservationCreated      = "OBSERVATION_CREATED"
	AgentPhaseRunCompleted            = "RUN_COMPLETED"
	AgentPhaseRunFailed               = "RUN_FAILED"
	AgentPhaseRunCancelled            = "RUN_CANCELLED"
)

type ToolExecutionStatus string

const (
	ToolExecPending     ToolExecutionStatus = "PENDING"
	ToolExecRunning     ToolExecutionStatus = "RUNNING"
	ToolExecCompleted   ToolExecutionStatus = "COMPLETED"
	ToolExecFailed      ToolExecutionStatus = "FAILED"
	ToolExecRejected    ToolExecutionStatus = "REJECTED"
	ToolExecUnavailable ToolExecutionStatus = "UNAVAILABLE"
)

type AnalysisRun struct {
	ID                      primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID          primitive.ObjectID  `bson:"organizationId"`
	CreatedByUserID         primitive.ObjectID  `bson:"createdByUserId"`
	ProductID               primitive.ObjectID  `bson:"productId"`
	MarketplaceImportRunID  *primitive.ObjectID `bson:"marketplaceImportRunId,omitempty"`
	ClientRequestID         string              `bson:"clientRequestId"`
	Status                  AnalysisRunStatus   `bson:"status"`
	CurrentPhase            string              `bson:"currentPhase,omitempty"`
	TraceID                 string              `bson:"traceId"`
	CorrelationID           string              `bson:"correlationId"`
	ConfigSnapshotID        primitive.ObjectID  `bson:"configSnapshotId"`
	ToolRegistryVersion     string              `bson:"toolRegistryVersion"`
	ToolPolicyVersion       string              `bson:"toolPolicyVersion"`
	CompliancePolicyVersion string              `bson:"compliancePolicyVersion"`
	PlannerVersion          string              `bson:"plannerVersion"`
	ObservationSchemaVersion string             `bson:"observationSchemaVersion"`
	IterationCount          int                 `bson:"iterationCount"`
	RetryCount              int                 `bson:"retryCount"`
	CancellationRequested   bool                `bson:"cancellationRequested"`
	CancelledByUserID       *primitive.ObjectID `bson:"cancelledByUserId,omitempty"`
	RecoveryAttempt         int                 `bson:"recoveryAttempt,omitempty"`
	LeaseOwnerID            string              `bson:"leaseOwnerId,omitempty"`
	LastHeartbeatAt         *time.Time          `bson:"lastHeartbeatAt,omitempty"`
	TerminalError           string              `bson:"terminalError,omitempty"`
	TerminalReason          string              `bson:"terminalReason,omitempty"`
	StartedAt               *time.Time          `bson:"startedAt,omitempty"`
	CompletedAt             *time.Time          `bson:"completedAt,omitempty"`
	CreatedAt               time.Time           `bson:"createdAt"`
	UpdatedAt               time.Time           `bson:"updatedAt"`
}

type AgentRunEvent struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organizationId"`
	RunID          primitive.ObjectID `bson:"runId"`
	Sequence       int64              `bson:"sequence"`
	Phase          string             `bson:"phase"`
	ToolName       string             `bson:"toolName,omitempty"`
	Status         AgentEventStatus   `bson:"status"`
	Metadata       map[string]string  `bson:"metadata,omitempty"`
	TraceID        string             `bson:"traceId"`
	Timestamp      time.Time          `bson:"timestamp"`
}

type ToolExecution struct {
	ID                  primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID      primitive.ObjectID  `bson:"organizationId"`
	AnalysisRunID       primitive.ObjectID  `bson:"analysisRunId"`
	ToolName            string              `bson:"toolName"`
	ToolVersion         string              `bson:"toolVersion"`
	Attempt             int                 `bson:"attempt"`
	AuthorizationDecision string            `bson:"authorizationDecision"`
	InputHash           string              `bson:"inputHash"`
	OutputHash          string              `bson:"outputHash,omitempty"`
	GrantNonceHash      string              `bson:"grantNonceHash,omitempty"`
	Status              ToolExecutionStatus `bson:"status"`
	ErrorCode           string              `bson:"errorCode,omitempty"`
	ErrorClass          string              `bson:"errorClass,omitempty"`
	TimeoutAt           *time.Time          `bson:"timeoutAt,omitempty"`
	AuditRef            string              `bson:"auditRef,omitempty"`
	StartedAt           *time.Time          `bson:"startedAt,omitempty"`
	CompletedAt         *time.Time          `bson:"completedAt,omitempty"`
	CreatedAt           time.Time           `bson:"createdAt"`
}

type ConfigSnapshot struct {
	ID                      primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID          *primitive.ObjectID `bson:"organizationId,omitempty"`
	SnapshotKind            string             `bson:"snapshotKind"`
	ToolRegistryVersion     string             `bson:"toolRegistryVersion"`
	ToolPolicyVersion       string             `bson:"toolPolicyVersion"`
	CompliancePolicyVersion string             `bson:"compliancePolicyVersion"`
	PlannerVersion          string             `bson:"plannerVersion"`
	ObservationSchemaVersion string            `bson:"observationSchemaVersion"`
	RuntimeConfig           map[string]any     `bson:"runtimeConfig,omitempty"`
	PublishedBy             primitive.ObjectID `bson:"publishedBy"`
	PublishedAt             time.Time          `bson:"publishedAt"`
	Reason                  string             `bson:"reason"`
}

const (
	ConfigSnapshotKindPhase5 = "PHASE5_AGENT_CORE"
	EventAnalysisRunStarted  SecurityEventType = "ANALYSIS_RUN_STARTED"
	EventAnalysisRunFailed   SecurityEventType = "ANALYSIS_RUN_FAILED"
	EventUnauthorizedTool    SecurityEventType = "UNAUTHORIZED_TOOL"
	EventAnalysisRunCancelled SecurityEventType = "ANALYSIS_RUN_CANCELLED"
)
