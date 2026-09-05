package importpkg

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
)

const (
	AmazonBabyMaxReviewsPerProduct = 100
	AmazonBabyDatasetSource        = "KAGGLE_AMAZON_BABY_ROOPALIK"
	AmazonBabyProductIDPrefix      = "amazonbaby_product_"
	AmazonBabyReviewIDPrefix       = "amazonbaby_review_"
	AmazonBabySourceURL            = "https://www.kaggle.com/datasets/roopalik/amazon-baby-dataset"
	AmazonBabyMinReviewLen         = 10
	AmazonBabyMaxReviewLen         = 10000
)

var ErrUnsupportedAmazonBabyFormat = errors.New("unsupported amazon baby csv format")

// DatasetLocalProductID returns a deterministic dataset-local product key.
// This is NOT an Amazon ASIN, GTIN, or native marketplace product ID.
func DatasetLocalProductID(normalizedName string) string {
	sum := sha256.Sum256([]byte(normalizedName))
	return AmazonBabyProductIDPrefix + hex.EncodeToString(sum[:8])
}

// DatasetLocalReviewID returns a deterministic dataset-local review key.
// This is NOT a source-native review ID.
func DatasetLocalReviewID(productKey string, rating *float64, reviewText string) string {
	ratingPart := "na"
	if rating != nil {
		ratingPart = fmt.Sprintf("%.1f", *rating)
	}
	normalized := normalize.NormalizeReviewText(reviewText)
	payload := fmt.Sprintf("%s|%s|%s", productKey, ratingPart, normalized)
	sum := sha256.Sum256([]byte(payload))
	return AmazonBabyReviewIDPrefix + hex.EncodeToString(sum[:8])
}

func cleanWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func boundReviewText(text string) string {
	if len(text) > AmazonBabyMaxReviewLen {
		return text[:AmazonBabyMaxReviewLen]
	}
	return text
}

func redactReviewPII(text string) (string, domain.PIIStatus) {
	if compliance.DetectPII(text) {
		return compliance.RedactPII(text), domain.PIIRedacted
	}
	return text, domain.PIIClean
}

type amazonBabyRow struct {
	name       string
	reviewText string
	rating     *float64
}

func parseAmazonBabyRows(records [][]string) ([]amazonBabyRow, ParseStats, error) {
	stats := ParseStats{RawRows: len(records) - 1}
	if len(records) < 2 {
		return nil, stats, ErrEmptyInput
	}

	header := indexHeader(records[0])
	nameKey := firstHeader(header, "name", "title", "product_title", "product name")
	reviewKey := firstHeader(header, "review", "review_text", "review_body", "text", "reviewbody")
	ratingKey := firstHeader(header, "rating", "star_rating", "stars", "score")
	if nameKey == "" || reviewKey == "" {
		return nil, stats, ErrUnsupportedAmazonBabyFormat
	}

	var rows []amazonBabyRow
	for _, row := range records[1:] {
		name := cleanWhitespace(cell(header, row, nameKey))
		reviewText := cleanWhitespace(cell(header, row, reviewKey))
		if name == "" || reviewText == "" {
			stats.RejectedRows++
			continue
		}
		if len(reviewText) < AmazonBabyMinReviewLen {
			stats.RejectedRows++
			continue
		}
		reviewText = boundReviewText(reviewText)

		var rating *float64
		if ratingKey != "" {
			if raw := cell(header, row, ratingKey); raw != "" {
				if v, err := parseRating(raw); err == nil {
					rating = &v
				} else {
					stats.RejectedRows++
					continue
				}
			}
		}
		rows = append(rows, amazonBabyRow{name: name, reviewText: reviewText, rating: rating})
		stats.AcceptedRows++
	}
	if len(rows) == 0 {
		return nil, stats, ErrEmptyInput
	}
	return rows, stats, nil
}

func buildAmazonBabyProducts(rows []amazonBabyRow, stats *ParseStats) []ProductRow {
	type acc struct {
		name    string
		reviews []ReviewRow
	}
	byProduct := map[string]*acc{}
	seenReviewIDs := map[string]struct{}{}
	seenFingerprints := map[string]struct{}{}

	for _, row := range rows {
		productKey := NormalizeProductKey(row.name)
		sourceProductID := DatasetLocalProductID(productKey)
		item, ok := byProduct[sourceProductID]
		if !ok {
			item = &acc{name: row.name}
			byProduct[sourceProductID] = item
		}
		if len(item.reviews) >= AmazonBabyMaxReviewsPerProduct {
			continue
		}

		sourceReviewID := DatasetLocalReviewID(sourceProductID, row.rating, row.reviewText)
		if _, dup := seenReviewIDs[sourceReviewID]; dup {
			stats.DuplicateRows++
			continue
		}
		text, piiStatus := redactReviewPII(row.reviewText)
		fp := normalize.ReviewFingerprint(domain.MarketplaceOther, sourceProductID, sourceReviewID, text)
		if _, dup := seenFingerprints[fp]; dup {
			stats.DuplicateRows++
			continue
		}
		seenReviewIDs[sourceReviewID] = struct{}{}
		seenFingerprints[fp] = struct{}{}

		lang := normalize.DetectLanguage(text)
		signals := DeriveReviewSignals(text, row.rating)
		governance := AmazonBabyDefaultGovernance()

		item.reviews = append(item.reviews, ReviewRow{
			ReviewText:     text,
			OriginalText:   row.reviewText,
			Rating:         row.rating,
			SourceReviewID: sourceReviewID,
			Language:       lang,
			PIIStatus:      piiStatus,
			Governance:     &governance,
			Signals:        &signals,
			Fingerprint:    fp,
		})
	}

	keys := make([]string, 0, len(byProduct))
	for k := range byProduct {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	products := make([]ProductRow, 0, len(keys))
	for _, sourceProductID := range keys {
		item := byProduct[sourceProductID]
		products = append(products, ProductRow{
			Name:            item.name,
			SourceProductID: sourceProductID,
			Reviews:         item.reviews,
		})
	}
	stats.UniqueProducts = len(products)
	return products
}

// ParseAmazonBabyCSV ingests Kaggle-style Amazon Baby review CSVs.
func ParseAmazonBabyCSV(reader io.Reader) (*ParseResult, error) {
	r := csv.NewReader(reader)
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	rows, stats, err := parseAmazonBabyRows(records)
	if err != nil {
		return nil, err
	}
	products := buildAmazonBabyProducts(rows, &stats)
	if len(products) == 0 {
		return nil, ErrEmptyInput
	}
	return &ParseResult{
		Products:   products,
		AccessMode: domain.AccessCSVImport,
		Source:     AmazonBabyDatasetSource,
		Stats:      stats,
	}, nil
}

func firstHeader(header map[string]int, keys ...string) string {
	for _, key := range keys {
		if _, ok := header[strings.ToLower(key)]; ok {
			return key
		}
	}
	return ""
}

func NormalizeProductKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func parseRating(raw string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(strings.TrimSpace(raw), "%f", &v)
	if err != nil {
		return 0, err
	}
	if v < 1 || v > 5 {
		return 0, fmt.Errorf("rating out of range")
	}
	return v, nil
}

// DetectAmazonBabyCSV returns true when header looks like Amazon Baby review CSV.
func DetectAmazonBabyCSV(header []string) bool {
	h := indexHeader(header)
	nameOK := firstHeader(h, "name", "title", "product_title", "product name") != ""
	reviewOK := firstHeader(h, "review", "review_text", "review_body", "text", "reviewbody") != ""
	return nameOK && reviewOK
}

type canonicalImportRecord struct {
	RecordType      string `json:"recordType"`
	Source          string `json:"source"`
	SourceType      string `json:"sourceType"`
	SourceProductID string `json:"sourceProductId"`
	SourceReviewID  string `json:"sourceReviewId"`
	Product         struct {
		Name     string  `json:"name"`
		Brand    *string `json:"brand"`
		Category *string `json:"category"`
	} `json:"product"`
	Review struct {
		Rating   *float64 `json:"rating"`
		Text     string   `json:"text"`
		Language string   `json:"language"`
	} `json:"review"`
	Signals    *ReviewSignals    `json:"signals"`
	Governance *ReviewGovernance `json:"governance"`
}

// ParseMiyunaJSONL reads canonical Miyuna JSONL import records.
func ParseMiyunaJSONL(reader io.Reader) (*ParseResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	stats := ParseStats{}
	byProduct := map[string]*ProductRow{}
	seenReviewIDs := map[string]struct{}{}
	seenFingerprints := map[string]struct{}{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		stats.RawRows++
		var rec canonicalImportRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			stats.RejectedRows++
			continue
		}
		if rec.Source != "" && rec.Source != AmazonBabyDatasetSource {
			stats.RejectedRows++
			continue
		}
		name := cleanWhitespace(rec.Product.Name)
		text := cleanWhitespace(rec.Review.Text)
		sourceProductID := strings.TrimSpace(rec.SourceProductID)
		if text == "" || len(text) < AmazonBabyMinReviewLen {
			stats.RejectedRows++
			continue
		}
		if sourceProductID == "" && name == "" {
			stats.RejectedRows++
			continue
		}
		text = boundReviewText(text)
		if sourceProductID == "" {
			sourceProductID = DatasetLocalProductID(NormalizeProductKey(name))
		}
		sourceReviewID := rec.SourceReviewID
		if sourceReviewID == "" {
			sourceReviewID = DatasetLocalReviewID(sourceProductID, rec.Review.Rating, text)
		}
		if _, dup := seenReviewIDs[sourceReviewID]; dup {
			stats.DuplicateRows++
			continue
		}

		redacted, piiStatus := redactReviewPII(text)
		if rec.Review.Language != "" {
			// trust provided language tag when present
		}
		lang := domain.LanguageTag(rec.Review.Language)
		if lang == "" {
			lang = normalize.DetectLanguage(redacted)
		}
		fp := normalize.ReviewFingerprint(domain.MarketplaceOther, sourceProductID, sourceReviewID, redacted)
		if _, dup := seenFingerprints[fp]; dup {
			stats.DuplicateRows++
			continue
		}
		seenReviewIDs[sourceReviewID] = struct{}{}
		seenFingerprints[fp] = struct{}{}

		governance := AmazonBabyDefaultGovernance()
		if rec.Governance != nil {
			governance = *rec.Governance
		}
		governance = enforceAmazonBabyGovernance(governance)

		signals := DeriveReviewSignals(redacted, rec.Review.Rating)
		if rec.Signals != nil {
			signals = *rec.Signals
		}

		prod, ok := byProduct[sourceProductID]
		if !ok {
			byProduct[sourceProductID] = &ProductRow{
				Name:            name,
				SourceProductID: sourceProductID,
				Reviews:         nil,
			}
			prod = byProduct[sourceProductID]
		}
		if len(prod.Reviews) >= AmazonBabyMaxReviewsPerProduct {
			continue
		}
		prod.Reviews = append(prod.Reviews, ReviewRow{
			ReviewText:     redacted,
			OriginalText:   text,
			Rating:         rec.Review.Rating,
			SourceReviewID: sourceReviewID,
			Language:       lang,
			PIIStatus:      piiStatus,
			Governance:     &governance,
			Signals:        &signals,
			Fingerprint:    fp,
		})
		stats.AcceptedRows++
	}

	if len(byProduct) == 0 {
		return nil, ErrEmptyInput
	}
	keys := make([]string, 0, len(byProduct))
	for k := range byProduct {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	products := make([]ProductRow, 0, len(keys))
	for _, k := range keys {
		products = append(products, *byProduct[k])
	}
	stats.UniqueProducts = len(products)
	return &ParseResult{
		Products:   products,
		AccessMode: domain.AccessJSONImport,
		Source:     AmazonBabyDatasetSource,
		Stats:      stats,
	}, nil
}

func enforceAmazonBabyGovernance(g ReviewGovernance) ReviewGovernance {
	g.ProvenanceStatus = domain.ProvenancePartial
	g.LicenseStatus = domain.LicenseUnknown
	g.UsageRightsStatus = domain.UsageRightsUnknown
	g.DatasetEligibility = domain.EligibilityQuarantined
	g.TrainingAllowed = false
	g.EvaluationAllowed = false
	return g
}
