package agent

import (
	"fmt"
	"strings"
)

type ToolIOSchema struct {
	InputRequired  []string
	OutputRequired []string
	OutputStatuses []string
}

var toolIOSchemas = map[string]ToolIOSchema{
	"policy_evaluator": {
		OutputRequired: []string{"allowed", "policyVersion", "profile"},
		OutputStatuses: []string{"OK"},
	},
	"compliance_checker": {
		OutputRequired: []string{"allowed", "policyVersion"},
		OutputStatuses: []string{"OK"},
	},
	"fetch_policy_checker": {
		OutputStatuses: []string{"OK", "SKIPPED"},
	},
	"review_sampler": {
		OutputRequired: []string{"totalAvailable", "sampled", "reviewIds", "sourceType"},
		OutputStatuses: []string{"OK"},
	},
	"dataset_validator": {
		OutputRequired: []string{"marketplaceReviewCount", "ugcCount", "eligibilitySummary"},
		OutputStatuses: []string{"OK"},
	},
	"pii_redactor": {
		OutputRequired: []string{"redactedMetadata"},
		OutputStatuses: []string{"OK"},
	},
	"evidence_validator": {
		InputRequired:  []string{"claimId", "claimText", "evidenceIds"},
		OutputRequired: []string{"claimId", "status", "evidenceIds", "issues"},
		OutputStatuses: []string{"OK"},
	},
	"safety_analyzer": {
		InputRequired:  []string{"analysisRunId", "productId", "evidenceIds"},
		OutputRequired: []string{"findings", "issues"},
		OutputStatuses: []string{"OK"},
	},
}

func ValidateToolOutput(toolName string, status string, payload map[string]any) error {
	schema, ok := toolIOSchemas[toolName]
	if !ok {
		return fmt.Errorf("%w: unknown tool schema", ErrInvalidInput)
	}
	if len(schema.OutputStatuses) > 0 && !containsString(schema.OutputStatuses, status) {
		return fmt.Errorf("%w: invalid output status %s", ErrSchemaValidation, status)
	}
	for _, field := range schema.OutputRequired {
		if _, ok := payload[field]; !ok {
			return fmt.Errorf("%w: missing output field %s", ErrSchemaValidation, field)
		}
	}
	return nil
}

func ValidateSafetyAnalyzerInput(input ToolInput) error {
	if input.RunID.IsZero() {
		return fmt.Errorf("%w: missing analysisRunId", ErrSchemaValidation)
	}
	if input.ProductID.IsZero() {
		return fmt.Errorf("%w: missing productId", ErrSchemaValidation)
	}
	if input.EvidenceIDs == nil {
		return fmt.Errorf("%w: missing evidenceIds", ErrSchemaValidation)
	}
	return nil
}

func ValidateEvidenceValidatorInput(input ToolInput) error {
	if strings.TrimSpace(input.ClaimID) == "" {
		return fmt.Errorf("%w: missing claimId", ErrSchemaValidation)
	}
	if strings.TrimSpace(input.ClaimText) == "" {
		return fmt.Errorf("%w: missing claimText", ErrSchemaValidation)
	}
	if input.EvidenceIDs == nil {
		return fmt.Errorf("%w: missing evidenceIds", ErrSchemaValidation)
	}
	return nil
}

func ValidateToolInputHash(toolName, inputHash string) error {
	if strings.TrimSpace(inputHash) == "" || len(inputHash) != 64 {
		return fmt.Errorf("%w: invalid input hash", ErrSchemaValidation)
	}
	if _, ok := toolIOSchemas[toolName]; !ok {
		return ErrToolUnavailable
	}
	return nil
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
