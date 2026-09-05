package normalize

import (
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

type ReviewInput struct {
	ReviewText     string
	Rating         *float64
	ReviewDate     *time.Time
	SourceReviewID string
}

// NormalizeReviewText applies deterministic whitespace and punctuation normalization.
func NormalizeReviewText(text string) string {
	text = strings.TrimSpace(text)
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}

// NormalizeReviewInput returns normalized review fields and detected language.
func NormalizeReviewInput(input ReviewInput) (string, domain.LanguageTag) {
	text := NormalizeReviewText(input.ReviewText)
	return text, DetectLanguage(text)
}
