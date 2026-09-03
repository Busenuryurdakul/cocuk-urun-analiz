package compliance

import (
	"regexp"
	"strings"
)

var (
	emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phonePattern = regexp.MustCompile(`(?:\+?\d[\d\s\-()]{7,}\d)`)
)

func containsPII(value string) bool {
	return emailPattern.MatchString(value) || phonePattern.MatchString(value)
}

func DetectPII(value string) bool {
	return containsPII(value)
}

func RedactPII(value string) string {
	out := emailPattern.ReplaceAllString(value, "[REDACTED_EMAIL]")
	out = phonePattern.ReplaceAllString(out, "[REDACTED_PHONE]")
	return out
}

func RedactFields(fields map[string]string) map[string]string {
	out := make(map[string]string, len(fields))
	for k, v := range fields {
		if ClassifyField(k, v) == ClassPII || ClassifyField(k, v) == ClassSensitive {
			out[k] = RedactPII(v)
		} else {
			out[k] = v
		}
	}
	return out
}

func SanitizeForAudit(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if DetectPII(v) {
		return "[REDACTED]"
	}
	if len(v) > 64 {
		return v[:64] + "…"
	}
	return v
}
