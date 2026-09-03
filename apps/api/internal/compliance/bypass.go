package compliance

import (
	"strings"
)

var bypassPatterns = []string{
	"disable compliance",
	"ignore gdpr",
	"ignore kvkk",
	"compliance off",
	"profile off",
	"bypass compliance",
	"skip compliance",
}

func DetectBypassAttempt(values ...string) bool {
	for _, v := range values {
		lower := strings.ToLower(strings.TrimSpace(v))
		for _, p := range bypassPatterns {
			if strings.Contains(lower, p) {
				return true
			}
		}
		if lower == "off" || lower == "disable" || lower == "none" {
			return true
		}
	}
	return false
}
