package analysis

const (
	WeightEvidenceQuality  = 0.22
	WeightEvidenceCount    = 0.12
	WeightSourceAgreement  = 0.14
	WeightSourceFreshness  = 0.08
	WeightProductMatch     = 0.12
	WeightReviewerValid    = 0.12
	WeightDataCompleteness = 0.10
	WeightReviewCoverage   = 0.10

	HighEvidenceQuality  = 0.85
	WeakEvidenceQuality  = 0.45
	StrongEvidenceCount  = 3
	WeakEvidenceCount    = 1
	StrongReviewCoverage = 8
	WeakReviewCoverage   = 2

	OverconfidenceMarkers = "kesinlikle güvenli|definitely safe|no risk|%100 güvenli|absolutely safe"
)
