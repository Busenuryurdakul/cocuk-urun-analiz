package compliance

import (
	"regexp"
	"strings"
)

var defaultForbiddenClaims = []*regexp.Regexp{
	regexp.MustCompile(`(?i)certified safe`),
	regexp.MustCompile(`(?i)guaranteed compliant`),
	regexp.MustCompile(`(?i)definitely safe`),
	regexp.MustCompile(`(?i)regulatory approval`),
	regexp.MustCompile(`(?i)legally guaranteed`),
}

func ValidateOutputText(text string, extraPatterns []string) []string {
	var violations []string
	for _, re := range defaultForbiddenClaims {
		if re.MatchString(text) {
			violations = append(violations, re.String())
		}
	}
	for _, p := range extraPatterns {
		if strings.TrimSpace(p) == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(p))
		if err != nil {
			continue
		}
		if re.MatchString(text) {
			violations = append(violations, p)
		}
	}
	return violations
}
