package importpkg_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
)

func TestParseAmazonBabyCSV(t *testing.T) {
	payload := `name,review,rating
Planetwise Flannel Wipes,Soft and absorbent wipe,4
Planetwise Flannel Wipes,Not durable enough for daily use,2
Annas Dream Quilt,Very warm and soft quilt,5
`
	result, err := importpkg.ParseAmazonBabyCSV(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(result.Products))
	}
	totalReviews := 0
	for _, p := range result.Products {
		totalReviews += len(p.Reviews)
		if !strings.HasPrefix(p.SourceProductID, importpkg.AmazonBabyProductIDPrefix) {
			t.Fatalf("expected dataset-local product id prefix, got %s", p.SourceProductID)
		}
	}
	if totalReviews != 3 {
		t.Fatalf("expected 3 reviews total, got %d", totalReviews)
	}
	if result.AccessMode != domain.AccessCSVImport {
		t.Fatalf("expected CSV_IMPORT, got %s", result.AccessMode)
	}
	if result.Source != importpkg.AmazonBabyDatasetSource {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	g := result.Products[0].Reviews[0].Governance
	if g == nil || g.DatasetEligibility != domain.EligibilityQuarantined {
		t.Fatal("expected QUARANTINED governance default")
	}
}

func TestDatasetLocalIDsStable(t *testing.T) {
	key := importpkg.NormalizeProductKey("Planetwise Flannel Wipes")
	id1 := importpkg.DatasetLocalProductID(key)
	id2 := importpkg.DatasetLocalProductID(key)
	if id1 != id2 {
		t.Fatalf("product id not stable: %s vs %s", id1, id2)
	}
	rating := 4.0
	rev1 := importpkg.DatasetLocalReviewID(id1, &rating, "Soft and absorbent wipe")
	rev2 := importpkg.DatasetLocalReviewID(id1, &rating, "Soft and absorbent wipe")
	if rev1 != rev2 {
		t.Fatalf("review id not stable")
	}
	if !strings.HasPrefix(rev1, importpkg.AmazonBabyReviewIDPrefix) {
		t.Fatalf("expected review prefix, got %s", rev1)
	}
}

func TestAmazonBabyDedup(t *testing.T) {
	payload := `name,review,rating
Toy,Great product for babies,5
Toy,Great product for babies,5
`
	result, err := importpkg.ParseAmazonBabyCSV(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Stats.DuplicateRows != 1 {
		t.Fatalf("expected 1 duplicate, got %d", result.Stats.DuplicateRows)
	}
	if len(result.Products[0].Reviews) != 1 {
		t.Fatalf("expected 1 unique review")
	}
}

func TestAmazonBabyPIIRedaction(t *testing.T) {
	payload := `name,review,rating
Toy,Contact me at user@example.com please,5
`
	result, err := importpkg.ParseAmazonBabyCSV(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rev := result.Products[0].Reviews[0]
	if rev.PIIStatus != domain.PIIRedacted {
		t.Fatalf("expected REDACTED pii status, got %s", rev.PIIStatus)
	}
	if strings.Contains(rev.ReviewText, "user@example.com") {
		t.Fatal("email should be redacted")
	}
}

func TestAmazonBabyRejectsShortReview(t *testing.T) {
	payload := `name,review,rating
Toy,short,5
`
	_, err := importpkg.ParseAmazonBabyCSV(strings.NewReader(payload))
	if err == nil {
		t.Fatal("expected error for unusable short review")
	}
}

func TestParseCSVDetectsAmazonBaby(t *testing.T) {
	payload := "name,review,rating\nToy,Great product for babies,5\n"
	result, err := importpkg.ParseCSV(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if result.Source != importpkg.AmazonBabyDatasetSource {
		t.Fatalf("expected amazon baby source, got %s", result.Source)
	}
}

func TestParseMiyunaJSONL(t *testing.T) {
	payload := `{"recordType":"MARKETPLACE_REVIEW","source":"KAGGLE_AMAZON_BABY_ROOPALIK","sourceType":"MARKETPLACE_REVIEW","sourceProductId":"amazonbaby_product_abc","sourceReviewId":"amazonbaby_review_def","product":{"name":"Toy"},"review":{"rating":5,"text":"Great product for babies","language":"EN"},"governance":{"provenanceStatus":"PARTIAL","licenseStatus":"UNKNOWN","usageRightsStatus":"UNKNOWN","datasetEligibility":"QUARANTINED"}}
`
	result, err := importpkg.ParseMiyunaJSONL(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse jsonl: %v", err)
	}
	if len(result.Products) != 1 || len(result.Products[0].Reviews) != 1 {
		t.Fatalf("unexpected parse result")
	}
	if result.Products[0].Reviews[0].Governance.DatasetEligibility != domain.EligibilityQuarantined {
		t.Fatal("jsonl import must remain quarantined")
	}
}

func TestParseMiyunaJSONLAllowsMissingProductNameWithASIN(t *testing.T) {
	payload := `{"recordType":"MARKETPLACE_REVIEW","source":"KAGGLE_AMAZON_BABY_ROOPALIK","sourceType":"MARKETPLACE_REVIEW","sourceProductId":"097293751X","sourceReviewId":"amazonbaby_review_def","product":{"name":null},"review":{"rating":5,"text":"Great product for babies with enough length","language":"EN"},"governance":{"provenanceStatus":"PARTIAL","licenseStatus":"UNKNOWN","usageRightsStatus":"UNKNOWN","datasetEligibility":"QUARANTINED"}}
`
	result, err := importpkg.ParseMiyunaJSONL(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse jsonl: %v", err)
	}
	if len(result.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(result.Products))
	}
	if result.Products[0].SourceProductID != "097293751X" {
		t.Fatalf("expected ASIN source product id, got %q", result.Products[0].SourceProductID)
	}
	if result.Products[0].Name != "" {
		t.Fatalf("expected missing product name, got %q", result.Products[0].Name)
	}
}

func TestMaxReviewsPerProductBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("name,review,rating\n")
	for i := 0; i < 105; i++ {
		b.WriteString(fmt.Sprintf("Toy,This is review number %d with enough length,5\n", i))
	}
	result, err := importpkg.ParseAmazonBabyCSV(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Products[0].Reviews) > importpkg.AmazonBabyMaxReviewsPerProduct {
		t.Fatalf("analysis sample exceeded max reviews per product")
	}
}
