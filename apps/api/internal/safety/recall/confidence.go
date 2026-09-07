package recall

const (
	ConfidenceExactSourceID     = 0.99
	ConfidenceExactGTIN         = 0.98
	ConfidenceExactUPC          = 0.97
	ConfidenceExactEAN          = 0.97
	ConfidenceManufacturerModel = 0.94
	ConfidenceBrandModel        = 0.93
	ConfidenceBrandName         = 0.78
	ConfidenceFuzzyName         = 0.42
	ConfidenceNone              = 0.0
	ConfirmedMatchMinConfidence = 0.90
	FuzzyNameOnlyConfidence     = 0.42
	MinReportedMatchConfidence  = 0.35
)
