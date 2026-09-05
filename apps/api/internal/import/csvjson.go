package importpkg

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

var ErrEmptyInput = errors.New("empty import payload")

type ProductRow struct {
	Name            string
	Brand           string
	Category        string
	Description     string
	TargetAge       string
	Materials       string
	SafetyWarnings  string
	CurrentPrice    any
	Currency        string
	SKU             string
	SourceProductID string
	Reviews         []ReviewRow
}

type ReviewRow struct {
	ReviewText     string
	OriginalText   string
	Rating         *float64
	SourceReviewID string
	Language       domain.LanguageTag
	PIIStatus      domain.PIIStatus
	Governance     *ReviewGovernance
	Signals        *ReviewSignals
	Fingerprint    string
}

type ParseStats struct {
	RawRows        int
	AcceptedRows   int
	RejectedRows   int
	DuplicateRows  int
	UniqueProducts int
}

type ParseResult struct {
	Products   []ProductRow
	AccessMode domain.AccessMode
	Source     string
	Stats      ParseStats
}

func ParseCSV(reader io.Reader) (*ParseResult, error) {
	r := csv.NewReader(reader)
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, ErrEmptyInput
	}
	if DetectAmazonBabyCSV(records[0]) {
		return ParseAmazonBabyCSV(strings.NewReader(renderCSV(records)))
	}
	header := indexHeader(records[0])
	var products []ProductRow
	for _, row := range records[1:] {
		if len(row) == 0 {
			continue
		}
		products = append(products, ProductRow{
			Name:            cell(header, row, "name", "title"),
			Brand:           cell(header, row, "brand"),
			Category:        cell(header, row, "category"),
			Description:     cell(header, row, "description"),
			TargetAge:       cell(header, row, "target_age", "targetAge"),
			Materials:       cell(header, row, "materials"),
			SafetyWarnings:  cell(header, row, "safety_warnings", "safetyWarnings"),
			Currency:        cell(header, row, "currency"),
			SKU:             cell(header, row, "sku"),
			SourceProductID: cell(header, row, "source_product_id", "sourceProductId"),
		})
	}
	return &ParseResult{Products: products, AccessMode: domain.AccessCSVImport}, nil
}

func ParseJSON(reader io.Reader) (*ParseResult, error) {
	var payload struct {
		Products []struct {
			Name            string      `json:"name"`
			Brand           string      `json:"brand"`
			Category        string      `json:"category"`
			Description     string      `json:"description"`
			TargetAge       string      `json:"targetAge"`
			Materials       string      `json:"materials"`
			SafetyWarnings  string      `json:"safetyWarnings"`
			CurrentPrice    any         `json:"currentPrice"`
			Currency        string      `json:"currency"`
			SKU             string      `json:"sku"`
			SourceProductID string      `json:"sourceProductId"`
			Reviews         []ReviewRow `json:"reviews"`
		} `json:"products"`
	}
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Products) == 0 {
		return nil, ErrEmptyInput
	}
	var products []ProductRow
	for _, p := range payload.Products {
		products = append(products, ProductRow{
			Name:            p.Name,
			Brand:           p.Brand,
			Category:        p.Category,
			Description:     p.Description,
			TargetAge:       p.TargetAge,
			Materials:       p.Materials,
			SafetyWarnings:  p.SafetyWarnings,
			CurrentPrice:    p.CurrentPrice,
			Currency:        p.Currency,
			SKU:             p.SKU,
			SourceProductID: p.SourceProductID,
			Reviews:         p.Reviews,
		})
	}
	return &ParseResult{Products: products, AccessMode: domain.AccessJSONImport}, nil
}

func NormalizeProductRow(row ProductRow, source domain.MarketplaceSource, sourceRecordID string) domain.Product {
	return normalize.NormalizeProductInput(normalize.ProductInput{
		Name:           row.Name,
		Brand:          row.Brand,
		Category:       row.Category,
		Description:    row.Description,
		TargetAge:      row.TargetAge,
		Materials:      row.Materials,
		SafetyWarnings: row.SafetyWarnings,
		CurrentPrice:   row.CurrentPrice,
		Currency:       row.Currency,
		SKU:            row.SKU,
	}, string(source), sourceRecordID)
}

func indexHeader(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		out[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return out
}

func cell(header map[string]int, row []string, keys ...string) string {
	for _, key := range keys {
		if idx, ok := header[strings.ToLower(key)]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
	}
	return ""
}

func NormalizeReviewRow(row ReviewRow, source domain.MarketplaceSource, sourceProductID string) (string, domain.LanguageTag, string) {
	if row.Fingerprint != "" && row.Language != "" {
		return row.ReviewText, row.Language, row.Fingerprint
	}
	text, lang := normalize.NormalizeReviewInput(normalize.ReviewInput{
		ReviewText:     row.ReviewText,
		Rating:         row.Rating,
		SourceReviewID: row.SourceReviewID,
	})
	if row.Language != "" {
		lang = row.Language
	}
	fp := row.Fingerprint
	if fp == "" {
		fp = normalize.ReviewFingerprint(source, sourceProductID, row.SourceReviewID, text)
	}
	return text, lang, fp
}

func Summary(result *ParseResult) string {
	if result == nil {
		return "empty"
	}
	if result.Source != "" {
		return fmt.Sprintf("%d products via %s (%s)", len(result.Products), result.AccessMode, result.Source)
	}
	return fmt.Sprintf("%d products via %s", len(result.Products), result.AccessMode)
}

// LooksLikeMiyunaJSONL detects line-delimited canonical Miyuna import records.
func LooksLikeMiyunaJSONL(content []byte) bool {
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return false
	}
	firstLine := trimmed
	if idx := strings.Index(trimmed, "\n"); idx >= 0 {
		firstLine = trimmed[:idx]
	}
	return strings.Contains(firstLine, `"recordType"`) && strings.Contains(firstLine, `"sourceProductId"`)
}

func renderCSV(records [][]string) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	for _, row := range records {
		_ = w.Write(row)
	}
	w.Flush()
	return b.String()
}
