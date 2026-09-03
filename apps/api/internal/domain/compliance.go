package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ComplianceProfile string

const (
	ProfileKVKK ComplianceProfile = "KVKK"
	ProfileGDPR ComplianceProfile = "GDPR"
	ProfileBoth ComplianceProfile = "BOTH"
)

type PolicyStatus string

const (
	PolicyStatusDraft     PolicyStatus = "DRAFT"
	PolicyStatusPublished PolicyStatus = "PUBLISHED"
	PolicyStatusArchived  PolicyStatus = "ARCHIVED"
)

type ConsentPurpose string

const (
	ConsentPurposeRegistration   ConsentPurpose = "REGISTRATION"
	ConsentPurposeOrgMembership  ConsentPurpose = "ORG_MEMBERSHIP"
	ConsentPurposeDataProcessing ConsentPurpose = "DATA_PROCESSING"
)

type ConsentSource string

const (
	ConsentSourceWeb   ConsentSource = "WEB"
	ConsentSourceAdmin ConsentSource = "ADMIN"
	ConsentSourceAPI   ConsentSource = "API"
)

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "PENDING"
	InvitationAccepted InvitationStatus = "ACCEPTED"
	InvitationRevoked  InvitationStatus = "REVOKED"
	InvitationExpired  InvitationStatus = "EXPIRED"
)

const (
	EventOrgCreated              SecurityEventType = "ORG_CREATED"
	EventOrgMemberInvited        SecurityEventType = "ORG_MEMBER_INVITED"
	EventOrgMemberJoined         SecurityEventType = "ORG_MEMBER_JOINED"
	EventOrgMemberRemoved        SecurityEventType = "ORG_MEMBER_REMOVED"
	EventOrgRoleChanged          SecurityEventType = "ORG_ROLE_CHANGED"
	EventComplianceProfileChange SecurityEventType = "COMPLIANCE_PROFILE_CHANGED"
	EventComplianceBypassAttempt SecurityEventType = "COMPLIANCE_BYPASS_ATTEMPT"
	EventCompliancePolicyPublish SecurityEventType = "COMPLIANCE_POLICY_PUBLISHED"
	EventInvitationReplay        SecurityEventType = "INVITATION_REPLAY"
	EventConsentGranted          SecurityEventType = "CONSENT_GRANTED"
	EventConsentWithdrawn        SecurityEventType = "CONSENT_WITHDRAWN"
)

type PolicyRules struct {
	RequireConsentPurposes []string `bson:"requireConsentPurposes"`
	ForbiddenClaimPatterns []string `bson:"forbiddenClaimPatterns"`
}

type CompliancePolicyVersion struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID *primitive.ObjectID `bson:"organizationId,omitempty"`
	Profile        ComplianceProfile   `bson:"profile"`
	Version        string              `bson:"version"`
	Status         PolicyStatus        `bson:"status"`
	EffectiveAt    time.Time           `bson:"effectiveAt"`
	Rules          PolicyRules         `bson:"rules"`
	Reason         string              `bson:"reason,omitempty"`
	CreatedBy      primitive.ObjectID  `bson:"createdBy"`
	CreatedAt      time.Time           `bson:"createdAt"`
	PublishedAt    *time.Time          `bson:"publishedAt,omitempty"`
}

type OrganizationInvitation struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID  `bson:"organizationId"`
	Email          string              `bson:"email"`
	Role           OrgRole             `bson:"role"`
	InviterID      primitive.ObjectID  `bson:"inviterId"`
	TokenHash      string              `bson:"tokenHash"`
	Status         InvitationStatus    `bson:"status"`
	ExpiresAt      time.Time           `bson:"expiresAt"`
	AcceptedAt     *time.Time          `bson:"acceptedAt,omitempty"`
	AcceptedBy     *primitive.ObjectID `bson:"acceptedByUserId,omitempty"`
	CreatedAt      time.Time           `bson:"createdAt"`
}

type Consent struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	UserID         primitive.ObjectID  `bson:"userId"`
	OrganizationID *primitive.ObjectID `bson:"organizationId,omitempty"`
	Purpose        ConsentPurpose      `bson:"purpose"`
	PolicyVersion  string              `bson:"policyVersion"`
	LawfulBasis    string              `bson:"lawfulBasis,omitempty"`
	GrantedAt      time.Time           `bson:"grantedAt"`
	WithdrawnAt    *time.Time          `bson:"withdrawnAt,omitempty"`
	Source         ConsentSource       `bson:"source"`
	AuditRequestID string              `bson:"auditRequestId,omitempty"`
}

type ComplianceEvent struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID *primitive.ObjectID `bson:"organizationId,omitempty"`
	UserID         *primitive.ObjectID `bson:"userId,omitempty"`
	EventType      string              `bson:"eventType"`
	RuleID         string              `bson:"ruleId,omitempty"`
	Result         string              `bson:"result"`
	Details        map[string]string   `bson:"details"`
	Timestamp      time.Time           `bson:"timestamp"`
}

type ConfigAuditLog struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	ChangedBy  primitive.ObjectID  `bson:"changedBy"`
	ChangedAt  time.Time           `bson:"changedAt"`
	Reason     string              `bson:"reason"`
	OldVersion string              `bson:"oldVersion"`
	NewVersion string              `bson:"newVersion"`
	FieldDiff  map[string]string   `bson:"fieldDiff"`
	OrgID      *primitive.ObjectID `bson:"organizationId,omitempty"`
}
