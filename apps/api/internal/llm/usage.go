package llm

import "strings"

// IsUpgradeOrFallback reports whether a persisted call used a model fallback
// or a quality escalation. Task type names such as deep_analysis are not counted
// by themselves; only an actual upgrade marker or fallback flag is.
func IsUpgradeOrFallback(fallbackUsed bool, routingReason string) bool {
	if fallbackUsed {
		return true
	}
	lower := strings.ToLower(routingReason)
	return strings.Contains(lower, "quality_escalation") || strings.Contains(lower, "escalat")
}
