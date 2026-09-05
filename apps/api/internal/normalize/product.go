package normalize

import (
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

type ProductInput struct {
	Name           string
	Brand          string
	Category       string
	Description    string
	TargetAge      string
	Materials      string
	SafetyWarnings string
	CurrentPrice   any
	OriginalPrice  any
	Currency       string
	Seller         string
	Rating         any
	ReviewCount    any
	SKU            string
	StockStatus    string
	Source         string
	SourceRecordID string
}

func fieldMeta(value any, source, sourceRecordID string) domain.ProductFieldMeta {
	return domain.ProductFieldMeta{
		Value:          value,
		Missing:        isMissingValue(value),
		Source:         source,
		SourceRecordID: sourceRecordID,
		ExtractedAt:    ptrTime(time.Now().UTC()),
	}
}

func isMissingValue(value any) bool {
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

func ptrTime(t time.Time) *time.Time {
	return &t
}

// NormalizeProductInput converts raw input into a domain Product with deterministic field normalization.
func NormalizeProductInput(input ProductInput, source, sourceRecordID string) domain.Product {
	name := strings.TrimSpace(input.Name)
	brand := strings.ToLower(strings.TrimSpace(input.Brand))
	category := strings.TrimSpace(input.Category)
	description := strings.TrimSpace(input.Description)

	return domain.Product{
		Name:           fieldMeta(name, source, sourceRecordID),
		Brand:          fieldMeta(brand, source, sourceRecordID),
		Category:       fieldMeta(category, source, sourceRecordID),
		Description:    fieldMeta(description, source, sourceRecordID),
		TargetAge:      fieldMeta(strings.TrimSpace(input.TargetAge), source, sourceRecordID),
		Materials:      fieldMeta(strings.TrimSpace(input.Materials), source, sourceRecordID),
		SafetyWarnings: fieldMeta(strings.TrimSpace(input.SafetyWarnings), source, sourceRecordID),
		CurrentPrice:   fieldMeta(input.CurrentPrice, source, sourceRecordID),
		OriginalPrice:  fieldMeta(input.OriginalPrice, source, sourceRecordID),
		Currency:       fieldMeta(strings.ToUpper(strings.TrimSpace(input.Currency)), source, sourceRecordID),
		Seller:         fieldMeta(strings.TrimSpace(input.Seller), source, sourceRecordID),
		Rating:         fieldMeta(input.Rating, source, sourceRecordID),
		ReviewCount:    fieldMeta(input.ReviewCount, source, sourceRecordID),
		SKU:            fieldMeta(strings.TrimSpace(input.SKU), source, sourceRecordID),
		StockStatus:    fieldMeta(strings.TrimSpace(input.StockStatus), source, sourceRecordID),
	}
}

// NormalizeIdentifier trims and uppercases product identifiers for dedup keys.
func NormalizeIdentifier(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}
