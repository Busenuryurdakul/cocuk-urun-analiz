package agent

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashVersion(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func RegistryVersion(reg *Registry) string {
	return HashVersion(reg.CanonicalContent())
}

func PlannerRulesVersion() string {
	rules := []byte(`phase5-deterministic-planner-v1
step1:policy_evaluator
step2:compliance_checker
step3:review_sampler_if_reviews
step4:dataset_validator
step5:pii_redactor_metadata
exclude:import_planner,ecommerce_fetcher,product_normalizer,analyzers,report,evidence
`)
	return HashVersion(rules)
}

func ToolPolicyVersion(reg *Registry) string {
	var b []byte
	for _, t := range reg.All() {
		line := t.Name + ":" + string(t.Availability) + ":" + string(t.SandboxClass) + "\n"
		b = append(b, []byte(line)...)
	}
	return HashVersion(b)
}

func ObservationSchemaVersion() string {
	schema := []byte(`observation-v1{productId,organizationId,reviewSampleCount,ugcCount,eligibilitySummary,redactedMetadata}`)
	return HashVersion(schema)
}
