package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrgType string

const (
	OrgTypePersonal     OrgType = "PERSONAL"
	OrgTypeOrganization OrgType = "ORGANIZATION"
)

type OrgRole string

const (
	RoleOwner   OrgRole = "OWNER"
	RoleAdmin   OrgRole = "ADMIN"
	RoleAnalyst OrgRole = "ANALYST"
	RoleViewer  OrgRole = "VIEWER"
)

type DevicePlatform string

const (
	DevicePlatformWeb DevicePlatform = "WEB"
)

type SecuritySeverity string

const (
	SeverityInfo     SecuritySeverity = "INFO"
	SeverityWarning  SecuritySeverity = "WARNING"
	SeverityCritical SecuritySeverity = "CRITICAL"
)

type SecurityEventType string

const (
	EventCrossTenantAccess  SecurityEventType = "CROSS_TENANT_ACCESS"
	EventAuthFailure        SecurityEventType = "AUTH_FAILURE"
	EventMFAFailure         SecurityEventType = "MFA_FAILURE"
	EventRefreshTokenReplay SecurityEventType = "REFRESH_TOKEN_REPLAY"
	EventOTPAttemptLimit    SecurityEventType = "OTP_ATTEMPT_LIMIT"
)

type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Email         string             `bson:"email"`
	PasswordHash  string             `bson:"passwordHash"`
	EmailVerified bool               `bson:"emailVerified"`
	MFAEnabled    bool               `bson:"mfaEnabled"`
	MFASecret     string             `bson:"mfaSecret,omitempty"`
	PersonalOrgID primitive.ObjectID `bson:"personalOrgId"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
}

type Organization struct {
	ID                      primitive.ObjectID `bson:"_id,omitempty"`
	Name                    string             `bson:"name"`
	Type                    OrgType            `bson:"type"`
	ComplianceProfile       string             `bson:"complianceProfile"`
	CompliancePolicyVersion string             `bson:"compliancePolicyVersion"`
	OwnerID                 primitive.ObjectID `bson:"ownerId"`
	CreatedAt               time.Time          `bson:"createdAt"`
	UpdatedAt               time.Time          `bson:"updatedAt"`
}

type OrganizationMember struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organizationId"`
	UserID         primitive.ObjectID `bson:"userId"`
	Role           OrgRole            `bson:"role"`
	InvitedAt      time.Time          `bson:"invitedAt"`
	JoinedAt       time.Time          `bson:"joinedAt"`
}

type Device struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            primitive.ObjectID `bson:"userId"`
	DeviceFingerprint string             `bson:"deviceFingerprint"`
	Platform          DevicePlatform     `bson:"platform"`
	Verified          bool               `bson:"verified"`
	LastActiveAt      time.Time          `bson:"lastActiveAt"`
	CreatedAt         time.Time          `bson:"createdAt"`
}

type SecurityEvent struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	OrganizationID *primitive.ObjectID `bson:"organizationId,omitempty"`
	UserID         *primitive.ObjectID `bson:"userId,omitempty"`
	EventType      SecurityEventType   `bson:"eventType"`
	Severity       SecuritySeverity    `bson:"severity"`
	Details        map[string]string   `bson:"details"`
	Timestamp      time.Time           `bson:"timestamp"`
}

// Session stores refresh token hash and session metadata. Access auth uses short-lived JWT.
type Session struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	UserID           primitive.ObjectID `bson:"userId"`
	OrganizationID   primitive.ObjectID `bson:"organizationId"`
	DeviceID         primitive.ObjectID `bson:"deviceId"`
	FamilyID         primitive.ObjectID `bson:"familyId"`
	RefreshTokenHash string             `bson:"refreshTokenHash"`
	Revoked          bool               `bson:"revoked"`
	ExpiresAt        time.Time          `bson:"expiresAt"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
}

type RotatedRefreshToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	SessionID primitive.ObjectID `bson:"sessionId"`
	FamilyID  primitive.ObjectID `bson:"familyId"`
	TokenHash string             `bson:"tokenHash"`
	RotatedAt time.Time          `bson:"rotatedAt"`
	ExpiresAt time.Time          `bson:"expiresAt"`
}

type PendingAuth struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	TokenHash         string             `bson:"tokenHash"`
	UserID            primitive.ObjectID `bson:"userId"`
	DeviceFingerprint string             `bson:"deviceFingerprint"`
	Stage             string             `bson:"stage"`
	FailedAttempts    int                `bson:"failedAttempts"`
	Locked            bool               `bson:"locked"`
	ExpiresAt         time.Time          `bson:"expiresAt"`
	CreatedAt         time.Time          `bson:"createdAt"`
}

type EmailVerification struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	TokenHash      string             `bson:"tokenHash"`
	FailedAttempts int                `bson:"failedAttempts"`
	Locked         bool               `bson:"locked"`
	ExpiresAt      time.Time          `bson:"expiresAt"`
	CreatedAt      time.Time          `bson:"createdAt"`
}

type DeviceVerification struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	DeviceID       primitive.ObjectID `bson:"deviceId"`
	CodeHash       string             `bson:"codeHash"`
	FailedAttempts int                `bson:"failedAttempts"`
	Locked         bool               `bson:"locked"`
	ExpiresAt      time.Time          `bson:"expiresAt"`
	CreatedAt      time.Time          `bson:"createdAt"`
}

type MFASetupChallenge struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	TokenHash      string             `bson:"tokenHash"`
	Secret         string             `bson:"secret"`
	FailedAttempts int                `bson:"failedAttempts"`
	Locked         bool               `bson:"locked"`
	ExpiresAt      time.Time          `bson:"expiresAt"`
	CreatedAt      time.Time          `bson:"createdAt"`
}
