package recall

import "time"

const (
	SourceCPSC  = "CPSC"
	SourceGUBIS = "GUBIS"
)

type Record struct {
	Source         string
	SourceRecordID string
	Title          string
	Brand          string
	ProductName    string
	Model          string
	Manufacturer   string
	GTIN           string
	UPC            string
	EAN            string
	Hazard         string
	Remedy         string
	Description    string
	PublishedAt    time.Time
	Reference      string
	RawMetadata    map[string]any
}

type Identity struct {
	SourceIDs    []string
	Brand        string
	Manufacturer string
	Model        string
	Name         string
	GTIN         string
	UPC          string
	EAN          string
}

type MatchResult struct {
	Matched        bool
	Confidence     float64
	Method         string
	Reasons        []string
	RequiresReview bool
}

const (
	MethodSourceID          = "SOURCE_ID"
	MethodGTIN              = "GTIN"
	MethodUPC               = "UPC"
	MethodEAN               = "EAN"
	MethodManufacturerModel = "MANUFACTURER_MODEL"
	MethodBrandModel        = "BRAND_MODEL"
	MethodBrandName         = "BRAND_NAME"
	MethodFuzzyName         = "FUZZY_NAME"
	MethodNone              = "NONE"
)
