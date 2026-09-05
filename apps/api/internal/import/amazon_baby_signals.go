package importpkg

import (
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

const (
	SentimentNegative = "NEGATIVE"
	SentimentNeutral  = "NEUTRAL"
	SentimentPositive = "POSITIVE"

	LabelSourceRatingDerived = "RATING_DERIVED"
	LabelMethodRuleBased     = "RULE_BASED"
	LabelConfidenceLow       = "LOW"
	LabelConfidenceMedium    = "MEDIUM"
	LabelConfidenceHigh      = "HIGH"

	QualityPositive = "POSITIVE"
	QualityNegative = "NEGATIVE"
	QualityMixed    = "MIXED"
	QualityUnknown  = "UNKNOWN"

	LabelReviewUnreviewed     = "UNREVIEWED"
	LabelReviewRulePrelabeled = "RULE_PRELABELED"
	LabelReviewHumanReviewed  = "HUMAN_REVIEWED"
	LabelReviewExpertApproved = "EXPERT_APPROVED"
)

type ReviewSignals struct {
	Sentiment                string
	SentimentLabelSource     string
	IssueType                domain.IssueType
	IssueLabelMethod         string
	IssueLabelConfidence     string
	SafetyRelatedObservation *bool
	QualitySignal            string
	ShortSummary             *string
	LabelReviewStatus        string
}

type ReviewGovernance struct {
	ProvenanceStatus   domain.ProvenanceStatus
	LicenseStatus      domain.LicenseStatus
	UsageRightsStatus  domain.UsageRightsStatus
	DatasetEligibility domain.DatasetEligibility
	TrainingAllowed    bool
	EvaluationAllowed  bool
}

func AmazonBabyDefaultGovernance() ReviewGovernance {
	return ReviewGovernance{
		ProvenanceStatus:   domain.ProvenancePartial,
		LicenseStatus:      domain.LicenseUnknown,
		UsageRightsStatus:  domain.UsageRightsUnknown,
		DatasetEligibility: domain.EligibilityQuarantined,
		TrainingAllowed:    false,
		EvaluationAllowed:  false,
	}
}

func DeriveSentimentFromRating(rating *float64) (string, string) {
	if rating == nil {
		return SentimentNeutral, LabelSourceRatingDerived
	}
	switch {
	case *rating <= 2:
		return SentimentNegative, LabelSourceRatingDerived
	case *rating == 3:
		return SentimentNeutral, LabelSourceRatingDerived
	default:
		return SentimentPositive, LabelSourceRatingDerived
	}
}

var issueKeywordRules = []struct {
	issue domain.IssueType
	words []string
}{
	{domain.IssueDurability, []string{"durability", "durable", "last long", "wear out", "worn out", "fall apart"}},
	{domain.IssueBreakage, []string{"broke", "broken", "break", "crack", "cracked", "snapped", "broke during"}},
	{domain.IssueAgeSizeMismatch, []string{"too small", "too big", "age", "size", "fit", "mismatch"}},
	{domain.IssueMaterial, []string{"material", "fabric", "plastic", "chemical", "bpa", "latex"}},
	{domain.IssueOdor, []string{"odor", "odour", "smell", "stink", "fragrance"}},
	{domain.IssuePackaging, []string{"packaging", "package", "box", "damaged box"}},
	{domain.IssueUsability, []string{"hard to use", "difficult", "confusing", "usability", "awkward"}},
	{domain.IssueQuality, []string{"quality", "cheap", "poorly made", "defect", "flimsy"}},
}

var safetyKeywords = []string{
	"choking", "choke", "sharp edge", "sharp", "strangulation", "strangle",
	"burn", "overheat", "over heating", "detached part", "locking failure",
	"restraint failure", "unsafe", "hazard", "suffocation",
}

func DeriveIssueType(text string) (domain.IssueType, string, string) {
	lower := strings.ToLower(text)
	for _, rule := range issueKeywordRules {
		for _, word := range rule.words {
			if strings.Contains(lower, word) {
				conf := LabelConfidenceMedium
				if len(word) > 8 {
					conf = LabelConfidenceHigh
				}
				return rule.issue, LabelMethodRuleBased, conf
			}
		}
	}
	return domain.IssueOther, LabelMethodRuleBased, LabelConfidenceLow
}

func DeriveSafetyObservation(text string) *bool {
	lower := strings.ToLower(text)
	for _, word := range safetyKeywords {
		if strings.Contains(lower, word) {
			v := true
			return &v
		}
	}
	v := false
	return &v
}

func DeriveQualitySignal(sentiment string, issue domain.IssueType, rating *float64) string {
	hasIssue := issue != domain.IssueOther
	switch {
	case sentiment == SentimentPositive && !hasIssue:
		return QualityPositive
	case sentiment == SentimentNegative || hasIssue:
		if sentiment == SentimentPositive && hasIssue {
			return QualityMixed
		}
		return QualityNegative
	case rating == nil:
		return QualityUnknown
	default:
		return QualityMixed
	}
}

func DeriveReviewSignals(text string, rating *float64) ReviewSignals {
	sentiment, labelSource := DeriveSentimentFromRating(rating)
	issue, method, conf := DeriveIssueType(text)
	safety := DeriveSafetyObservation(text)
	quality := DeriveQualitySignal(sentiment, issue, rating)
	return ReviewSignals{
		Sentiment:                sentiment,
		SentimentLabelSource:     labelSource,
		IssueType:                issue,
		IssueLabelMethod:         method,
		IssueLabelConfidence:     conf,
		SafetyRelatedObservation: safety,
		QualitySignal:            quality,
		ShortSummary:             nil,
		LabelReviewStatus:        LabelReviewRulePrelabeled,
	}
}
