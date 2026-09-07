package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EvidenceSupportStatus string

const (
	EvidenceSupported          EvidenceSupportStatus = "SUPPORTED"
	EvidencePartiallySupported EvidenceSupportStatus = "PARTIALLY_SUPPORTED"
	EvidenceUnsupported        EvidenceSupportStatus = "UNSUPPORTED"
	EvidenceContradicted       EvidenceSupportStatus = "CONTRADICTED"
)

const (
	EvidenceIssueNoEvidence         = "NO_EVIDENCE"
	EvidenceIssueInvalidEvidenceID  = "INVALID_EVIDENCE_ID"
	EvidenceIssueWrongTenant        = "WRONG_TENANT"
	EvidenceIssueWrongAnalysisRun   = "WRONG_ANALYSIS_RUN"
	EvidenceIssueInvalidReliability = "INVALID_RELIABILITY"
	EvidenceIssueInvalidFreshness   = "INVALID_FRESHNESS"
	EvidenceIssueMissingSource      = "MISSING_SOURCE"
	EvidenceIssueMissingReference   = "MISSING_REFERENCE"
	EvidenceIssueClaimMismatch      = "CLAIM_MISMATCH"
	EvidenceIssueContradictory      = "CONTRADICTORY_EVIDENCE"
)

type Evidence struct {
	ID             primitive.ObjectID     `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID     `bson:"organizationId"`
	AnalysisRunID  primitive.ObjectID     `bson:"analysisRunId"`
	ProductID      primitive.ObjectID     `bson:"productId"`
	ClaimID        string                 `bson:"claimId,omitempty"`
	Source         string                 `bson:"source"`
	SourceType     string                 `bson:"sourceType"`
	Claim          string                 `bson:"claim"`
	Snippet        string                 `bson:"snippet,omitempty"`
	Reference      string                 `bson:"reference,omitempty"`
	Reliability    float64                `bson:"reliability"`
	Freshness      float64                `bson:"freshness"`
	RetrievedAt    time.Time              `bson:"retrievedAt"`
	CreatedAt      time.Time              `bson:"createdAt"`
	UpdatedAt      time.Time              `bson:"updatedAt"`
	Metadata       map[string]interface{} `bson:"metadata,omitempty"`
}

type EvidenceClaimValidation struct {
	ID             primitive.ObjectID    `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID    `bson:"organizationId"`
	AnalysisRunID  primitive.ObjectID    `bson:"analysisRunId"`
	ClaimID        string                `bson:"claimId"`
	ClaimText      string                `bson:"claimText"`
	EvidenceIDs    []primitive.ObjectID  `bson:"evidenceIds"`
	SupportStatus  EvidenceSupportStatus `bson:"supportStatus"`
	Issues         []string              `bson:"issues"`
	CreatedAt      time.Time             `bson:"createdAt"`
	UpdatedAt      time.Time             `bson:"updatedAt"`
}
