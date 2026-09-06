package llm

import "strings"

const (
	minPromptCharsForQualityEscalation = 350
	minResponseCharsWhenPromptLong     = 100
)

// ShouldEscalateQuality decides whether a fast-model answer should be retried on the heavy model.
func ShouldEscalateQuality(taskType, userPrompt, content string) bool {
	taskType = strings.TrimSpace(strings.ToLower(taskType))
	if taskType == "deep_analysis" || taskType == "review" {
		return false
	}

	prompt := strings.TrimSpace(userPrompt)
	output := strings.TrimSpace(content)
	if prompt == "" {
		return false
	}

	if len(prompt) >= minPromptCharsForQualityEscalation {
		return true
	}

	lower := strings.ToLower(prompt)
	complexityHints := []string{
		"adim adim", "adım adım", "detayli", "detaylı", "kapsamli", "kapsamlı", "uzun surec", "uzun süreç",
		"comprehensive", "step by step", "in depth", "compare", "risk analizi", "risk analysis",
		"multiple", "cok adim", "çok adım", "uzun cevap",
	}
	for _, hint := range complexityHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}

	if strings.Count(prompt, "?") >= 2 && len(prompt) >= 180 {
		return true
	}

	if len(prompt) >= 250 && len(output) < minResponseCharsWhenPromptLong {
		return true
	}

	return false
}
