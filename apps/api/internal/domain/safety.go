package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SafetyFindingType string

const (
	SafetyChoking              SafetyFindingType = "CHOKING"
	SafetySuffocation          SafetyFindingType = "SUFFOCATION"
	SafetyStrangulation        SafetyFindingType = "STRANGULATION"
	SafetyChemical             SafetyFindingType = "CHEMICAL"
	SafetyFire                 SafetyFindingType = "FIRE"
	SafetyElectrical           SafetyFindingType = "ELECTRICAL"
	SafetyStructural           SafetyFindingType = "STRUCTURAL"
	SafetyAgeSuitability       SafetyFindingType = "AGE_SUITABILITY"
	SafetyHygiene              SafetyFindingType = "HYGIENE"
	SafetyInjury               SafetyFindingType = "INJURY"
	SafetyRecall               SafetyFindingType = "RECALL"
	SafetyMisleadingClaim      SafetyFindingType = "MISLEADING_CLAIM"
	SafetyOther                SafetyFindingType = "OTHER"
	SafetyInsufficientEvidence SafetyFindingType = "INSUFFICIENT_EVIDENCE"
)

type SafetySeverity string

const (
	SafetySeverityLow      SafetySeverity = "LOW"
	SafetySeverityMedium   SafetySeverity = "MEDIUM"
	SafetySeverityHigh     SafetySeverity = "HIGH"
	SafetySeverityCritical SafetySeverity = "CRITICAL"
)

type SafetyFinding struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID   `bson:"organizationId"`
	AnalysisRunID  primitive.ObjectID   `bson:"analysisRunId"`
	ProductID      primitive.ObjectID   `bson:"productId"`
	Type           SafetyFindingType    `bson:"type"`
	Severity       SafetySeverity       `bson:"severity"`
	Confidence     float64              `bson:"confidence"`
	EvidenceIDs    []primitive.ObjectID `bson:"evidenceIds"`
	Rationale      string               `bson:"rationale"`
	Source         string               `bson:"source"`
	Issues         []string             `bson:"issues,omitempty"`
	CreatedAt      time.Time            `bson:"createdAt"`
}

type RecallMatchRecord struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organizationId"`
	AnalysisRunID  primitive.ObjectID `bson:"analysisRunId"`
	ProductID      primitive.ObjectID `bson:"productId"`
	Source         string             `bson:"source"`
	SourceRecordID string             `bson:"sourceRecordId"`
	Matched        bool               `bson:"matched"`
	Confidence     float64            `bson:"confidence"`
	Method         string             `bson:"method"`
	Reference      string             `bson:"reference,omitempty"`
	RequiresReview bool               `bson:"requiresReview"`
	Hazard         string             `bson:"hazard,omitempty"`
	Title          string             `bson:"title,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt"`
}

func ValidSafetyFindingType(v string) bool {
	switch SafetyFindingType(v) {
	case SafetyChoking, SafetySuffocation, SafetyStrangulation, SafetyChemical, SafetyFire,
		SafetyElectrical, SafetyStructural, SafetyAgeSuitability, SafetyHygiene, SafetyInjury,
		SafetyRecall, SafetyMisleadingClaim, SafetyOther, SafetyInsufficientEvidence:
		return true
	default:
		return false
	}
}
