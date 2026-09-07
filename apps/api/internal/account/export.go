package account

const ExportSchemaVersion = "1.0.0"

type ExportDocument struct {
	SchemaVersion    string                   `json:"schemaVersion"`
	ExportedAt       string                   `json:"exportedAt"`
	User             ExportUserProfile        `json:"user"`
	Consents         []ExportConsent          `json:"consents"`
	Memberships      []ExportMembership       `json:"memberships"`
	Devices          []ExportDevice           `json:"devices"`
	AnalysisActivity []ExportAnalysisActivity `json:"analysisActivity"`
	ActivityLog      []ExportActivityEntry    `json:"activityLog"`
}

type ExportUserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	MFAEnabled    bool   `json:"mfaEnabled"`
	PersonalOrgID string `json:"personalOrgId"`
	CreatedAt     string `json:"createdAt"`
}

type ExportConsent struct {
	Purpose        string  `json:"purpose"`
	PolicyVersion  string  `json:"policyVersion"`
	OrganizationID *string `json:"organizationId,omitempty"`
	GrantedAt      string  `json:"grantedAt"`
	WithdrawnAt    *string `json:"withdrawnAt,omitempty"`
}

type ExportMembership struct {
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	OrganizationType string `json:"organizationType"`
	Role             string `json:"role"`
	JoinedAt         string `json:"joinedAt"`
}

type ExportDevice struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	Label        string `json:"label"`
	Verified     bool   `json:"verified"`
	LastActiveAt string `json:"lastActiveAt"`
	CreatedAt    string `json:"createdAt"`
}

type ExportAnalysisActivity struct {
	RunID           string `json:"runId"`
	OrganizationID  string `json:"organizationId"`
	Status          string `json:"status"`
	ClientRequestID string `json:"clientRequestId,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type ExportActivityEntry struct {
	Action         string  `json:"action"`
	OrganizationID *string `json:"organizationId,omitempty"`
	Timestamp      string  `json:"timestamp"`
}
