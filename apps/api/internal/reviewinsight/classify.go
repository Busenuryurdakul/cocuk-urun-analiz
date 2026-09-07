package reviewinsight

import (
	"strings"
	"unicode"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

const MaxReviews = 100

type classifiedDoc struct {
	text     string
	positive bool
	negative bool
	kinds    map[domain.ReviewSignalKind]string
}

func classifyText(raw string, rating *float64) classifiedDoc {
	text := strings.ToLower(strings.TrimSpace(raw))
	doc := classifiedDoc{text: text, kinds: map[domain.ReviewSignalKind]string{}}
	if rating != nil {
		if *rating >= 4 {
			doc.positive = true
		} else if *rating <= 2 {
			doc.negative = true
		}
	}
	switch {
	case containsAny(text, "boğul", "chok", "small part", "küçük parça", "yut", "zehir", "chemical", "yanık", "strangul", "suffocat", "recall", "elektrik"):
		doc.kinds[domain.ReviewSignalSafety] = topicFor(text, "safety")
		doc.negative = true
	}
	if containsAny(text, "kalite", "broken", "kırık", "defective", "bozuk", "kalitesiz") {
		doc.kinds[domain.ReviewSignalQuality] = "quality"
		doc.negative = true
	}
	if containsAny(text, "dayanak", "kırıldı", "broke after", "kısa ömür", "durab") {
		doc.kinds[domain.ReviewSignalDurability] = "durability"
		doc.negative = true
	}
	if containsAny(text, "zor", "kullanım", "confus", "talimat yok", "kullanışsız") {
		doc.kinds[domain.ReviewSignalUsability] = "usability"
		doc.negative = true
	}
	if containsAny(text, "kolay", "kullanışlı", "ease of use", "pratik") && !containsAny(text, "zor", "kullanışsız") {
		doc.kinds[domain.ReviewSignalUsability] = "ease_of_use"
		doc.positive = true
	}
	if containsAny(text, "montaj", "assemble", "kurulum", "talimat") {
		doc.kinds[domain.ReviewSignalAssembly] = "assembly"
	}
	if containsAny(text, "yaş", "age", "çok küçük", "too young", "3 yaş") {
		doc.kinds[domain.ReviewSignalAge] = "age_mismatch"
		doc.negative = true
	}
	if containsAny(text, "harika", "güzel", "recommend", "tavsiye", "sturdy", "sağlam", "memnun") {
		doc.positive = true
	}
	if containsAny(text, "berbat", "kötü", "iade", "return", "pişman", "waste") {
		doc.negative = true
	}
	return doc
}

func containsAny(text string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}

func topicFor(text, fallback string) string {
	switch {
	case strings.Contains(text, "chok") || strings.Contains(text, "boğul") || strings.Contains(text, "yut"):
		return "choking"
	case strings.Contains(text, "recall"):
		return "recall"
	default:
		return fallback
	}
}

func normalizeTokens(s string) []string {
	parts := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) >= 4 {
			out = append(out, p)
		}
	}
	return out
}
