package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MarketplaceSource string

const (
	MarketplaceHepsiburada MarketplaceSource = "HEPSIBURADA"
	MarketplaceTrendyol    MarketplaceSource = "TRENDYOL"
	MarketplaceAmazon      MarketplaceSource = "AMAZON"
	MarketplaceN11         MarketplaceSource = "N11"
	MarketplaceOther       MarketplaceSource = "OTHER"
	MarketplaceMiyuna      MarketplaceSource = "MIYUNA"
)

type AccessMode string

const (
	AccessAuthorizedAPI    AccessMode = "AUTHORIZED_API"
	AccessPermittedPublic  AccessMode = "PERMITTED_PUBLIC_FETCH"
	AccessLicensedDataset  AccessMode = "LICENSED_DATASET"
	AccessCSVImport        AccessMode = "CSV_IMPORT"
	AccessJSONImport       AccessMode = "JSON_IMPORT"
	AccessUserProvidedFile AccessMode = "USER_PROVIDED_FILE"
	AccessManualEntry      AccessMode = "MANUAL_ENTRY"
)

type MatchStatus string

const (
	MatchVerified   MatchStatus = "VERIFIED"
	MatchPartial    MatchStatus = "PARTIAL"
	MatchUnverified MatchStatus = "UNVERIFIED"
)

type MatchMethod string

const (
	MatchIdentifier     MatchMethod = "IDENTIFIER"
	MatchExplicit       MatchMethod = "EXPLICIT"
	MatchManual         MatchMethod = "MANUAL"
	MatchFuzzyCandidate MatchMethod = "FUZZY_CANDIDATE"
)

type UsageStatus string

const (
	UsageUsing UsageStatus = "USING"
	UsageUsed  UsageStatus = "USED"
)

type SatisfactionLevel string

const (
	SatisfactionVerySatisfied   SatisfactionLevel = "VERY_SATISFIED"
	SatisfactionSatisfied       SatisfactionLevel = "SATISFIED"
	SatisfactionNeutral         SatisfactionLevel = "NEUTRAL"
	SatisfactionUnsatisfied     SatisfactionLevel = "UNSATISFIED"
	SatisfactionVeryUnsatisfied SatisfactionLevel = "VERY_UNSATISFIED"
)

type IssueType string

const (
	IssueDurability               IssueType = "DURABILITY"
	IssueBreakage                 IssueType = "BREAKAGE"
	IssueAgeSizeMismatch          IssueType = "AGE_SIZE_MISMATCH"
	IssueMaterial                 IssueType = "MATERIAL"
	IssueOdor                     IssueType = "ODOR"
	IssuePackaging                IssueType = "PACKAGING"
	IssueUsability                IssueType = "USABILITY"
	IssueQuality                  IssueType = "QUALITY"
	IssueSafetyRelatedObservation IssueType = "SAFETY_RELATED_OBSERVATION"
	IssueOther                    IssueType = "OTHER"
)

type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "PENDING"
	ModerationApproved ModerationStatus = "APPROVED"
	ModerationRejected ModerationStatus = "REJECTED"
)

type QualityStatus string

const (
	QualityPending    QualityStatus = "PENDING"
	QualityApproved   QualityStatus = "APPROVED"
	QualityLowQuality QualityStatus = "LOW_QUALITY"
	QualityRejected   QualityStatus = "REJECTED"
)

type PIIStatus string

const (
	PIIClean       PIIStatus = "CLEAN"
	PIIRedacted    PIIStatus = "REDACTED"
	PIIQuarantined PIIStatus = "QUARANTINED"
)

type SpamStatus string

const (
	SpamUnknown SpamStatus = "UNKNOWN"
	SpamClean   SpamStatus = "CLEAN"
	SpamSuspect SpamStatus = "SUSPECT"
)

type DuplicateStatus string

const (
	DuplicateUnique   DuplicateStatus = "UNIQUE"
	DuplicateDetected DuplicateStatus = "DUPLICATE"
)

type ProvenanceStatus string

const (
	ProvenanceVerified ProvenanceStatus = "VERIFIED"
	ProvenancePartial  ProvenanceStatus = "PARTIAL"
	ProvenanceUnknown  ProvenanceStatus = "UNKNOWN"
	ProvenanceRejected ProvenanceStatus = "REJECTED"
)

type LicenseStatus string

const (
	LicenseApproved   LicenseStatus = "APPROVED"
	LicenseRestricted LicenseStatus = "RESTRICTED"
	LicenseUnknown    LicenseStatus = "UNKNOWN"
	LicenseRejected   LicenseStatus = "REJECTED"
)

type UsageRightsStatus string

const (
	UsageRightsApproved   UsageRightsStatus = "APPROVED"
	UsageRightsRestricted UsageRightsStatus = "RESTRICTED"
	UsageRightsUnknown    UsageRightsStatus = "UNKNOWN"
	UsageRightsRejected   UsageRightsStatus = "REJECTED"
)

type DatasetEligibility string

const (
	EligibilityTrainingApproved DatasetEligibility = "TRAINING_APPROVED"
	EligibilityEvalOnly         DatasetEligibility = "EVAL_ONLY"
	EligibilityAnalysisOnly     DatasetEligibility = "ANALYSIS_ONLY"
	EligibilityQuarantined      DatasetEligibility = "QUARANTINED"
	EligibilityRejected         DatasetEligibility = "REJECTED"
)

type DatasetRecordType string

const (
	RecordMarketplaceReview DatasetRecordType = "MARKETPLACE_REVIEW"
	RecordMiyunaUGC         DatasetRecordType = "MIYUNA_UGC"
	RecordProduct           DatasetRecordType = "PRODUCT"
	RecordOfficialSource    DatasetRecordType = "OFFICIAL_SOURCE"
	RecordOther             DatasetRecordType = "OTHER"
)

type DatasetVersionStatus string

const (
	DatasetDraft DatasetVersionStatus = "DRAFT"
)

type DatasetSplit string

const (
	SplitTrain       DatasetSplit = "TRAIN"
	SplitValidation  DatasetSplit = "VALIDATION"
	SplitHeldOutEval DatasetSplit = "HELD_OUT_EVAL"
)

type MarketplaceImportStatus string

const (
	ImportPending          MarketplaceImportStatus = "PENDING"
	ImportRunning          MarketplaceImportStatus = "RUNNING"
	ImportSucceeded        MarketplaceImportStatus = "SUCCEEDED"
	ImportPartial          MarketplaceImportStatus = "PARTIAL"
	ImportFailed           MarketplaceImportStatus = "FAILED"
	ImportRejectedByPolicy MarketplaceImportStatus = "REJECTED_BY_POLICY"
)

type LanguageTag string

const (
	LanguageTR      LanguageTag = "TR"
	LanguageEN      LanguageTag = "EN"
	LanguageUnknown LanguageTag = "UNKNOWN"
)

type ProductFieldMeta struct {
	Value          any        `bson:"value,omitempty" json:"value"`
	Missing        bool       `bson:"missing" json:"missing"`
	MissingReason  string     `bson:"missingReason,omitempty" json:"missingReason,omitempty"`
	Source         string     `bson:"source,omitempty" json:"source,omitempty"`
	SourceRecordID string     `bson:"sourceRecordId,omitempty" json:"sourceRecordId,omitempty"`
	Confidence     float64    `bson:"confidence,omitempty" json:"confidence,omitempty"`
	ExtractedAt    *time.Time `bson:"extractedAt,omitempty" json:"extractedAt,omitempty"`
}

type Product struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organizationId"`
	Name           ProductFieldMeta   `bson:"name"`
	Brand          ProductFieldMeta   `bson:"brand"`
	Category       ProductFieldMeta   `bson:"category"`
	Description    ProductFieldMeta   `bson:"description"`
	TargetAge      ProductFieldMeta   `bson:"targetAge"`
	Materials      ProductFieldMeta   `bson:"materials"`
	SafetyWarnings ProductFieldMeta   `bson:"safetyWarnings"`
	CurrentPrice   ProductFieldMeta   `bson:"currentPrice"`
	OriginalPrice  ProductFieldMeta   `bson:"originalPrice"`
	Currency       ProductFieldMeta   `bson:"currency"`
	Seller         ProductFieldMeta   `bson:"seller"`
	Rating         ProductFieldMeta   `bson:"rating"`
	ReviewCount    ProductFieldMeta   `bson:"reviewCount"`
	Attributes     ProductFieldMeta   `bson:"attributes"`
	ImageRefs      ProductFieldMeta   `bson:"imageRefs"`
	StockStatus    ProductFieldMeta   `bson:"stockStatus"`
	SKU            ProductFieldMeta   `bson:"sku"`
	CreatedBy      primitive.ObjectID `bson:"createdBy"`
	CreatedAt      time.Time          `bson:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"`
}

type ProductSourceMapping struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID  primitive.ObjectID `bson:"organizationId"`
	ProductID       primitive.ObjectID `bson:"productId"`
	Source          MarketplaceSource  `bson:"source"`
	SourceProductID string             `bson:"sourceProductId"`
	SourceURL       string             `bson:"sourceUrl,omitempty"`
	GTIN            string             `bson:"gtin,omitempty"`
	EAN             string             `bson:"ean,omitempty"`
	UPC             string             `bson:"upc,omitempty"`
	SKU             string             `bson:"sku,omitempty"`
	Brand           string             `bson:"brand,omitempty"`
	Model           string             `bson:"model,omitempty"`
	MatchStatus     MatchStatus        `bson:"matchStatus"`
	MatchMethod     MatchMethod        `bson:"matchMethod"`
	Confidence      float64            `bson:"confidence,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
}

type UserExperience struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID     primitive.ObjectID `bson:"organizationId"`
	ProductID          primitive.ObjectID `bson:"productId"`
	UserID             primitive.ObjectID `bson:"userId"`
	UsageStatus        UsageStatus        `bson:"usageStatus"`
	SatisfactionLevel  SatisfactionLevel  `bson:"satisfactionLevel"`
	Rating             *int               `bson:"rating,omitempty"`
	IssueType          *IssueType         `bson:"issueType,omitempty"`
	Narrative          string             `bson:"narrative"`
	MarketplaceURL     string             `bson:"marketplaceUrl,omitempty"`
	SourceType         string             `bson:"sourceType"`
	ConsentRecordID    primitive.ObjectID `bson:"consentRecordId"`
	PIIStatus          PIIStatus          `bson:"piiStatus"`
	ModerationStatus   ModerationStatus   `bson:"moderationStatus"`
	QualityStatus      QualityStatus      `bson:"qualityStatus"`
	DatasetEligibility DatasetEligibility `bson:"datasetEligibility"`
	CreatedAt          time.Time          `bson:"createdAt"`
	UpdatedAt          time.Time          `bson:"updatedAt"`
}

type MarketplaceReview struct {
	ID                     primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID         primitive.ObjectID  `bson:"organizationId"`
	ProductID              primitive.ObjectID  `bson:"productId"`
	ProductSourceMappingID *primitive.ObjectID `bson:"productSourceMappingId,omitempty"`
	SourceType             string              `bson:"sourceType"`
	Source                 MarketplaceSource   `bson:"source"`
	SourceURL              string              `bson:"sourceUrl,omitempty"`
	SourceProductID        string              `bson:"sourceProductId,omitempty"`
	SourceReviewID         string              `bson:"sourceReviewId,omitempty"`
	Rating                 *float64            `bson:"rating,omitempty"`
	ReviewText             string              `bson:"reviewText"`
	ReviewDate             *time.Time          `bson:"reviewDate,omitempty"`
	FetchedAt              time.Time           `bson:"fetchedAt"`
	Language               LanguageTag         `bson:"language"`
	PIIStatus              PIIStatus           `bson:"piiStatus"`
	ModerationStatus       ModerationStatus    `bson:"moderationStatus"`
	SpamStatus             SpamStatus          `bson:"spamStatus"`
	DuplicateStatus        DuplicateStatus     `bson:"duplicateStatus"`
	QualityStatus          QualityStatus       `bson:"qualityStatus"`
	ProvenanceStatus       ProvenanceStatus    `bson:"provenanceStatus"`
	LicenseStatus          LicenseStatus       `bson:"licenseStatus"`
	UsageRightsStatus      UsageRightsStatus   `bson:"usageRightsStatus"`
	DatasetEligibility     DatasetEligibility  `bson:"datasetEligibility"`
	RawStorageRef          string              `bson:"rawStorageRef,omitempty"`
	Fingerprint            string              `bson:"fingerprint"`
	CreatedAt              time.Time           `bson:"createdAt"`
}

type MarketplaceImportRun struct {
	ID                 primitive.ObjectID      `bson:"_id,omitempty"`
	OrganizationID     primitive.ObjectID      `bson:"organizationId"`
	RequestedBy        primitive.ObjectID      `bson:"requestedBy"`
	Source             MarketplaceSource       `bson:"source"`
	SourceURL          string                  `bson:"sourceUrl,omitempty"`
	ImportRef          string                  `bson:"importRef,omitempty"`
	AccessMode         AccessMode              `bson:"accessMode"`
	Status             MarketplaceImportStatus `bson:"status"`
	StartedAt          *time.Time              `bson:"startedAt,omitempty"`
	FinishedAt         *time.Time              `bson:"finishedAt,omitempty"`
	RecordsSeen        int                     `bson:"recordsSeen"`
	RecordsAccepted    int                     `bson:"recordsAccepted"`
	RecordsRejected    int                     `bson:"recordsRejected"`
	RecordsQuarantined int                     `bson:"recordsQuarantined"`
	ErrorCode          string                  `bson:"errorCode,omitempty"`
	ErrorMessage       string                  `bson:"errorMessage,omitempty"`
	ProductID          *primitive.ObjectID     `bson:"productId,omitempty"`
	CreatedAt          time.Time               `bson:"createdAt"`
}

type RawSourcePayload struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID   primitive.ObjectID `bson:"organizationId"`
	Source           MarketplaceSource  `bson:"source"`
	SourceURL        string             `bson:"sourceUrl,omitempty"`
	FetchMode        AccessMode         `bson:"fetchMode"`
	FetchedAt        time.Time          `bson:"fetchedAt"`
	ContentType      string             `bson:"contentType"`
	ContentHash      string             `bson:"contentHash"`
	ObjectStorageRef string             `bson:"objectStorageRef"`
	SizeBytes        int64              `bson:"sizeBytes"`
	Status           string             `bson:"status"`
	ImportRunID      primitive.ObjectID `bson:"importRunId,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt"`
}

type DatasetRecord struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID     primitive.ObjectID  `bson:"organizationId"`
	RecordType         DatasetRecordType   `bson:"recordType"`
	SourceRecordID     primitive.ObjectID  `bson:"sourceRecordId"`
	ProductID          primitive.ObjectID  `bson:"productId"`
	DatasetEligibility DatasetEligibility  `bson:"datasetEligibility"`
	ProvenanceStatus   ProvenanceStatus    `bson:"provenanceStatus"`
	LicenseStatus      LicenseStatus       `bson:"licenseStatus"`
	UsageRightsStatus  UsageRightsStatus   `bson:"usageRightsStatus"`
	QualityStatus      QualityStatus       `bson:"qualityStatus"`
	PIIStatus          PIIStatus           `bson:"piiStatus"`
	DatasetVersionID   *primitive.ObjectID `bson:"datasetVersionId,omitempty"`
	Split              *DatasetSplit       `bson:"split,omitempty"`
	CreatedAt          time.Time           `bson:"createdAt"`
	UpdatedAt          time.Time           `bson:"updatedAt"`
}

type DatasetVersion struct {
	ID                      primitive.ObjectID   `bson:"_id,omitempty"`
	OrganizationID          primitive.ObjectID   `bson:"organizationId"`
	Version                 string               `bson:"version"`
	Status                  DatasetVersionStatus `bson:"status"`
	SourceCounts            map[string]int       `bson:"sourceCounts,omitempty"`
	SourceDistribution      map[string]int       `bson:"sourceDistribution,omitempty"`
	EligibilityDistribution map[string]int       `bson:"eligibilityDistribution,omitempty"`
	LanguageDistribution    map[string]int       `bson:"languageDistribution,omitempty"`
	NormalizerVersion       string               `bson:"normalizerVersion"`
	PIIPolicyVersion        string               `bson:"piiPolicyVersion"`
	LicensePolicyVersion    string               `bson:"licensePolicyVersion"`
	ContentHash             string               `bson:"contentHash,omitempty"`
	CreatedBy               primitive.ObjectID   `bson:"createdBy"`
	CreatedAt               time.Time            `bson:"createdAt"`
}

const (
	SourceTypeMarketplaceReview = "MARKETPLACE_REVIEW"
	SourceTypeMiyunaUGC         = "MIYUNA_UGC"
	NormalizerVersionPhase4     = "phase4-v1"
)

const (
	EventProductCreated          SecurityEventType = "PRODUCT_CREATED"
	EventUGCCreated              SecurityEventType = "UGC_CREATED"
	EventUGCUpdated              SecurityEventType = "UGC_UPDATED"
	EventUGCDeleted              SecurityEventType = "UGC_DELETED"
	EventMarketplaceImportQueued SecurityEventType = "MARKETPLACE_IMPORT_QUEUED"
	EventMarketplaceImportFailed SecurityEventType = "MARKETPLACE_IMPORT_FAILED"
	EventSSRFBlocked             SecurityEventType = "SSRF_BLOCKED"
	EventFetchPolicyViolation    SecurityEventType = "FETCH_POLICY_VIOLATION"
	EventDatasetDraftBuilt       SecurityEventType = "DATASET_DRAFT_BUILT"
)

const (
	DeferredFetchReason = "LIVE_MARKETPLACE_FETCH_DEFERRED_PENDING_AUTHORIZED_API"
)
