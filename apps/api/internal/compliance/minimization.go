package compliance

import "fmt"

var allowedFieldsByOperation = map[string]map[string]bool{
	"create_organization": {
		"name":              true,
		"complianceProfile": true,
	},
	"invite_member": {
		"email": true,
		"role":  true,
	},
	"update_compliance_profile": {
		"complianceProfile": true,
	},
	"grant_consent": {
		"purpose": true,
	},
	"create_user_experience": {
		"narrative":         true,
		"usageStatus":       true,
		"satisfactionLevel": true,
	},
}

func ValidateMinimization(operation string, fields map[string]string) error {
	allowed, ok := allowedFieldsByOperation[operation]
	if !ok {
		return nil
	}
	for key := range fields {
		if !allowed[key] {
			return fmt.Errorf("%w: unexpected field %s", ErrComplianceViolation, key)
		}
	}
	return nil
}
