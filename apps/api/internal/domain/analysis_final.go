package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FinalAnalysisSchemaVersion = "1.0.0"

type ProductDecision string

const (
	DecisionAllow            ProductDecision = "ALLOW"
	DecisionAllowWithWarning ProductDecision = "ALLOW_WITH_WARNING"
	DecisionReviewRequired   ProductDecision = "REVIEW_REQUIRED"
	DecisionBlock            ProductDecision = "BLOCK"
)

type HallucinationFlag string

const (
	FlagUnsupportedClaim        HallucinationFlag = "UNSUPPORTED_CLAIM"
	FlagInventedRecall          HallucinationFlag = "INVENTED_RECALL"
	FlagContradictoryEvidence   HallucinationFlag = "CONTRADICTORY_EVIDENCE"
	FlagInsufficientEvidence    HallucinationFlag = "INSUFFICIENT_EVIDENCE"
	FlagOverconfidentConclusion HallucinationFlag = "OVERCONFIDENT_CONCLUSION"
	FlagSourceMismatch          HallucinationFlag = "SOURCE_MISMATCH"
)

type ReviewSignalKind string

const (
	ReviewSignalPositive   ReviewSignalKind = "POSITIVE"
	ReviewSignalNegative   ReviewSignalKind = "NEGATIVE"
	ReviewSignalSafety     ReviewSignalKind = "SAFETY"
	ReviewSignalQuality    ReviewSignalKind = "QUALITY"
	ReviewSignalDurability ReviewSignalKind = "DURABILITY"
	ReviewSignalUsability  ReviewSignalKind = "USABILITY"
	ReviewSignalAssembly   ReviewSignalKind = "ASSEMBLY"
	ReviewSignalAge        ReviewSignalKind = "AGE"
	ReviewSignalRepeated   ReviewSignalKind = "REPEATED"
)

type ReviewSignal struct {
	Topic   string           `bson:"topic" json:"topic"`
	Count   int              `bson:"count" json:"count"`
	Summary string           `bson:"summary" json:"summary"`
	Kind    ReviewSignalKind `bson:"kind" json:"kind"`
}

type ReviewInsightRecord struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID     primitive.ObjectID `bson:"organizationId"`
	AnalysisRunID      primitive.ObjectID `bson:"analysisRunId"`
	ProductID          primitive.ObjectID `bson:"productId"`
	ReviewCount        int                `bson:"reviewCount"`
	PositiveSignals    []ReviewSignal     `bson:"positiveSignals"`
	NegativeSignals    []ReviewSignal     `bson:"negativeSignals"`
	SafetySignals      []ReviewSignal     `bson:"safetySignals"`
	QualityIssues      []ReviewSignal     `bson:"qualityIssues"`
	DurabilityIssues   []ReviewSignal     `bson:"durabilityIssues"`
	UsabilityIssues    []ReviewSignal     `bson:"usabilityIssues"`
	AssemblyIssues     []ReviewSignal     `bson:"assemblyIssues"`
	AgeMismatchSignals []ReviewSignal     `bson:"ageMismatchSignals"`
	RepeatedComplaints []ReviewSignal     `bson:"repeatedComplaints"`
	CreatedAt          time.Time          `bson:"createdAt"`
}

type AnalysisLLMResult struct {
	Role                 string    `bson:"role" json:"role"`
	Provider             string    `bson:"provider" json:"provider"`
	Model                string    `bson:"model" json:"model"`
	Persona              string    `bson:"persona" json:"persona"`
	RoutingPolicyVersion string    `bson:"routingPolicyVersion,omitempty" json:"routingPolicyVersion,omitempty"`
	ConfigSnapshotID     string    `bson:"configSnapshotId,omitempty" json:"configSnapshotId,omitempty"`
	Output               string    `bson:"output" json:"output"`
	Flags                []string  `bson:"flags,omitempty" json:"flags,omitempty"`
	CreatedAt            time.Time `bson:"createdAt" json:"createdAt"`
}

type FinalAnalysisResult struct {
	SchemaVersion      string              `bson:"schemaVersion" json:"schemaVersion"`
	Summary            string              `bson:"summary" json:"summary"`
	OverallRisk        string              `bson:"overallRisk" json:"overallRisk"`
	Confidence         float64             `bson:"confidence" json:"confidence"`
	ConfidenceFactors  map[string]float64  `bson:"confidenceFactors,omitempty" json:"confidenceFactors,omitempty"`
	PositiveSignals    []ReviewSignal      `bson:"positiveSignals" json:"positiveSignals"`
	NegativeSignals    []ReviewSignal      `bson:"negativeSignals" json:"negativeSignals"`
	ReviewInsights     []ReviewSignal      `bson:"reviewInsights" json:"reviewInsights"`
	SafetyFindings     []SafetyFinding     `bson:"safetyFindings,omitempty" json:"-"`
	Recalls            []RecallMatchRecord `bson:"recalls,omitempty" json:"-"`
	Decision           ProductDecision     `bson:"decision" json:"decision"`
	Recommendation     string              `bson:"recommendation" json:"recommendation"`
	Limitations        []string            `bson:"limitations" json:"limitations"`
	HallucinationFlags []string            `bson:"hallucinationFlags" json:"hallucinationFlags"`
	Worker             AnalysisLLMResult   `bson:"worker" json:"worker"`
	Reviewer           AnalysisLLMResult   `bson:"reviewer" json:"reviewer"`
	CreatedAt          time.Time           `bson:"createdAt" json:"createdAt"`
}
