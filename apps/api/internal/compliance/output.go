package compliance

import (
	"regexp"
	"strings"
)

const ForbiddenClaimReplacement = "[yasaklı güvenlik iddiası kaldırıldı]"

var defaultForbiddenClaims = []*regexp.Regexp{
	regexp.MustCompile(`(?i)certified safe`),
	regexp.MustCompile(`(?i)guaranteed compliant`),
	regexp.MustCompile(`(?i)definitely safe`),
	regexp.MustCompile(`(?i)regulatory approval`),
	regexp.MustCompile(`(?i)legally guaranteed`),
}

func SanitizeForbiddenClaims(text string, extraPatterns []string) (string, []string) {
	violations := ValidateOutputText(text, extraPatterns)
	if len(violations) == 0 {
		return text, nil
	}
	out := text
	for _, re := range defaultForbiddenClaims {
		out = re.ReplaceAllString(out, ForbiddenClaimReplacement)
	}
	for _, p := range extraPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(p))
		if err != nil {
			continue
		}
		out = re.ReplaceAllString(out, ForbiddenClaimReplacement)
	}
	return out, violations
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
