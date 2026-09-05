package normalize

import (
	"strings"
	"unicode"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

var turkishChars = "çğıöşüÇĞİÖŞÜ"

// DetectLanguage returns a deterministic language tag from text content.
func DetectLanguage(text string) domain.LanguageTag {
	text = strings.TrimSpace(text)
	if text == "" {
		return domain.LanguageUnknown
	}

	turkishScore := 0
	latinScore := 0
	for _, r := range text {
		if strings.ContainsRune(turkishChars, r) {
			turkishScore++
		}
		if unicode.IsLetter(r) && r <= unicode.MaxASCII {
			latinScore++
		}
	}
	if turkishScore > 0 {
		return domain.LanguageTR
	}
	if latinScore > 0 {
		return domain.LanguageEN
	}
	return domain.LanguageUnknown
}
