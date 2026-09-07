package recall

import (
	"strings"
	"unicode"
)

func Match(identity Identity, rec Record) MatchResult {
	id := normalizeIdentity(identity)
	recN := normalizeRecord(rec)
	if identityEmpty(id) {
		return MatchResult{Matched: false, Method: MethodNone, RequiresReview: true}
	}

	if containsNorm(id.SourceIDs, recN.SourceRecordID) && recN.SourceRecordID != "" {
		return confirmed(MethodSourceID, ConfidenceExactSourceID, "source_id_exact")
	}
	if bothEqual(id.GTIN, recN.GTIN) {
		return confirmed(MethodGTIN, ConfidenceExactGTIN, "gtin_exact")
	}
	if bothEqual(id.UPC, recN.UPC) {
		return confirmed(MethodUPC, ConfidenceExactUPC, "upc_exact")
	}
	if bothEqual(id.EAN, recN.EAN) {
		return confirmed(MethodEAN, ConfidenceExactEAN, "ean_exact")
	}

	brandMismatch := bothPresentDiffer(id.Brand, recN.Brand)
	modelMismatch := bothPresentDiffer(id.Model, recN.Model)
	if brandMismatch || modelMismatch {
		if fuzzyOnlyPossible(id, recN) && !brandMismatch {
			return reviewRequired(MethodFuzzyName, ConfidenceFuzzyName, "fuzzy_name_only")
		}
		return MatchResult{Matched: false, Method: MethodNone, Reasons: mismatchReasons(brandMismatch, modelMismatch), RequiresReview: true}
	}

	if bothEqual(id.Manufacturer, recN.Manufacturer) && bothEqual(id.Model, recN.Model) {
		return confirmed(MethodManufacturerModel, ConfidenceManufacturerModel, "manufacturer_exact", "model_exact")
	}
	if bothEqual(id.Brand, recN.Brand) && bothEqual(id.Model, recN.Model) {
		return confirmed(MethodBrandModel, ConfidenceBrandModel, "brand_exact", "model_exact")
	}
	if bothEqual(id.Brand, recN.Brand) && namesAligned(id.Name, recN.ProductName) && (id.Model == "" || recN.Model == "") {
		return reviewRequired(MethodBrandName, ConfidenceBrandName, "brand_exact", "name_normalized")
	}

	sameOrMissingBrand := id.Brand == recN.Brand || id.Brand == "" || recN.Brand == ""
	if fuzzyName(id.Name, recN.ProductName) && sameOrMissingBrand {
		if id.Brand != "" && recN.Brand != "" && id.Brand != recN.Brand {
			return MatchResult{Matched: false, Method: MethodNone, Reasons: []string{"brand_mismatch"}, RequiresReview: true}
		}
		return reviewRequired(MethodFuzzyName, ConfidenceFuzzyName, "fuzzy_name_only")
	}
	return MatchResult{Matched: false, Method: MethodNone, RequiresReview: true}
}

func Confirmed(result MatchResult) bool {
	return result.Matched && !result.RequiresReview && result.Confidence >= ConfirmedMatchMinConfidence
}

func confirmed(method string, confidence float64, reasons ...string) MatchResult {
	return MatchResult{Matched: true, Confidence: confidence, Method: method, Reasons: reasons, RequiresReview: false}
}

func reviewRequired(method string, confidence float64, reasons ...string) MatchResult {
	return MatchResult{Matched: confidence >= MinReportedMatchConfidence, Confidence: confidence, Method: method, Reasons: reasons, RequiresReview: true}
}

func normalizeIdentity(in Identity) Identity {
	ids := make([]string, 0, len(in.SourceIDs))
	for _, id := range in.SourceIDs {
		if n := normID(id); n != "" {
			ids = append(ids, n)
		}
	}
	return Identity{
		SourceIDs:    ids,
		Brand:        normText(in.Brand),
		Manufacturer: normText(in.Manufacturer),
		Model:        normID(in.Model),
		Name:         normText(in.Name),
		GTIN:         normID(in.GTIN),
		UPC:          normID(in.UPC),
		EAN:          normID(in.EAN),
	}
}

func normalizeRecord(in Record) Record {
	return Record{
		Source:         strings.ToUpper(strings.TrimSpace(in.Source)),
		SourceRecordID: normID(in.SourceRecordID),
		Title:          normText(in.Title),
		Brand:          normText(in.Brand),
		ProductName:    normText(in.ProductName),
		Model:          normID(in.Model),
		Manufacturer:   normText(in.Manufacturer),
		GTIN:           normID(in.GTIN),
		UPC:            normID(in.UPC),
		EAN:            normID(in.EAN),
	}
}

func identityEmpty(id Identity) bool {
	return len(id.SourceIDs) == 0 && id.Brand == "" && id.Manufacturer == "" && id.Model == "" && id.Name == "" && id.GTIN == "" && id.UPC == "" && id.EAN == ""
}

func bothEqual(a, b string) bool {
	return a != "" && b != "" && a == b
}

func bothPresentDiffer(a, b string) bool {
	return a != "" && b != "" && a != b
}

func namesAligned(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}

func fuzzyOnlyPossible(id Identity, rec Record) bool {
	return id.Name != "" && rec.ProductName != "" && fuzzyName(id.Name, rec.ProductName)
}

func fuzzyName(a, b string) bool {
	if a == "" || b == "" || a == b {
		return a != "" && a == b
	}
	ta, tb := tokens(a), tokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	overlap := 0
	for _, x := range ta {
		for _, y := range tb {
			if x == y || strings.HasPrefix(x, y) || strings.HasPrefix(y, x) {
				overlap++
				break
			}
		}
	}
	den := len(ta)
	if len(tb) > den {
		den = len(tb)
	}
	return float64(overlap)/float64(den) >= 0.6
}

func tokens(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) >= 2 {
			out = append(out, p)
		}
	}
	return out
}

func mismatchReasons(brand, model bool) []string {
	out := make([]string, 0, 2)
	if brand {
		out = append(out, "brand_mismatch")
	}
	if model {
		out = append(out, "model_mismatch")
	}
	return out
}

func containsNorm(list []string, v string) bool {
	if v == "" {
		return false
	}
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func normID(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func normText(v string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(v))), " ")
}
