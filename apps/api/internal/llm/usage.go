package llm

import "strings"

// QualityEscalationMarker is written into routingReason when a call is
// quality-escalated. Keep this in sync with CountUpgradeOrFallbackSince.
const QualityEscalationMarker = "quality_escalation=true"

// IsUpgradeOrFallback reports whether a persisted call used a model fallback
// or a quality escalation. Task type names such as deep_analysis are not counted
// by themselves; only an actual upgrade marker or fallback flag is.
func IsUpgradeOrFallback(fallbackUsed bool, routingReason string) bool {
	if fallbackUsed {
		return true
	}
	return strings.Contains(strings.ToLower(routingReason), QualityEscalationMarker)
}
