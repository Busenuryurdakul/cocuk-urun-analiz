package marketplace

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

// publicFieldSlugs maps internal canonical field keys to user-facing slug names.
var publicFieldSlugs = map[string]string{
	"name":          "name",
	"currentPrice":  "price",
	"originalPrice": "price",
	"rating":        "rating",
	"reviewCount":   "reviewCount",
	"stockStatus":   "availability",
	"brand":         "brand",
	"imageRefs":     "images",
}

type MergeOutcome struct {
	Product       domain.Product
	UpdatedFields []string
	FieldsChanged bool
}

func mergeProductFromFetch(current domain.Product, input normalize.ProductInput, source, sourceRecordID string, fetchedAt time.Time) MergeOutcome {
	incoming := normalize.NormalizeProductInput(input, source, sourceRecordID)
	out := current
	changedSlugs := map[string]struct{}{}

	apply := func(key string, dst *domain.ProductFieldMeta, src domain.ProductFieldMeta) {
		if !shouldApplyIncoming(*dst, src) {
			return
		}
		if fieldValuesEqual(dst.Value, src.Value) {
			return
		}
		*dst = src
		extracted := fetchedAt.UTC()
		dst.ExtractedAt = &extracted
		dst.Source = source
		dst.SourceRecordID = sourceRecordID
		dst.Missing = src.Missing
		if slug, ok := publicFieldSlugs[key]; ok {
			changedSlugs[slug] = struct{}{}
		}
	}

	apply("name", &out.Name, incoming.Name)
	apply("brand", &out.Brand, incoming.Brand)
	apply("category", &out.Category, incoming.Category)
	apply("description", &out.Description, incoming.Description)
	apply("targetAge", &out.TargetAge, incoming.TargetAge)
	apply("materials", &out.Materials, incoming.Materials)
	apply("safetyWarnings", &out.SafetyWarnings, incoming.SafetyWarnings)
	apply("currentPrice", &out.CurrentPrice, incoming.CurrentPrice)
	apply("originalPrice", &out.OriginalPrice, incoming.OriginalPrice)
	apply("currency", &out.Currency, incoming.Currency)
	apply("seller", &out.Seller, incoming.Seller)
	apply("rating", &out.Rating, incoming.Rating)
	apply("reviewCount", &out.ReviewCount, incoming.ReviewCount)
	apply("stockStatus", &out.StockStatus, incoming.StockStatus)
	apply("sku", &out.SKU, incoming.SKU)
	apply("imageRefs", &out.ImageRefs, incoming.ImageRefs)

	fields := make([]string, 0, len(changedSlugs))
	for slug := range changedSlugs {
		fields = append(fields, slug)
	}
	sortStrings(fields)
	return MergeOutcome{
		Product:       out,
		UpdatedFields: fields,
		FieldsChanged: len(fields) > 0,
	}
}

func shouldApplyIncoming(current, incoming domain.ProductFieldMeta) bool {
	if incoming.Missing || isEmptyValue(incoming.Value) {
		return false
	}
	if !isValidIncomingValue(incoming.Value) {
		return false
	}
	return true
}

func isValidIncomingValue(value any) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case float64:
		return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
	case int:
		return v >= 0
	case int64:
		return v >= 0
	default:
		if value == nil {
			return false
		}
		s := strings.TrimSpace(fmt.Sprint(value))
		if s == "" || s == "<nil>" {
			return false
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return !math.IsNaN(f) && f >= 0
		}
		return true
	}
}

func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

func fieldValuesEqual(a, b any) bool {
	if isEmptyValue(a) && isEmptyValue(b) {
		return true
	}
	if isEmptyValue(a) || isEmptyValue(b) {
		return false
	}
	na, errA := toFloat(a)
	nb, errB := toFloat(b)
	if errA == nil && errB == nil {
		return math.Abs(na-nb) < 0.0001
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(a)), strings.TrimSpace(fmt.Sprint(b)))
}

func toFloat(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	}
}

func sortStrings(items []string) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
