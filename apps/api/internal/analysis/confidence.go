package analysis

import (
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

type ConfidenceInput struct {
	Evidence       []domain.Evidence
	Validations    []domain.EvidenceClaimValidation
	Findings       []domain.SafetyFinding
	Matches        []domain.RecallMatchRecord
	ReviewCount    int
	HasIdentity    bool
	Hallucination  []string
	ReviewerPassed bool
}

type ConfidenceResult struct {
	Score   float64
	Factors map[string]float64
}

func ComputeConfidence(in ConfidenceInput) ConfidenceResult {
	quality := evidenceQuality(in.Evidence)
	count := clamp01(float64(len(in.Evidence)) / float64(StrongEvidenceCount))
	agreement := sourceAgreement(in.Validations, in.Evidence)
	freshness := sourceFreshness(in.Evidence)
	match := productMatch(in.Matches)
	reviewer := 0.55
	if in.ReviewerPassed {
		reviewer = 0.9
	}
	completeness := dataCompleteness(in)
	coverage := clamp01(float64(in.ReviewCount) / float64(StrongReviewCoverage))

	if !in.HasIdentity {
		match = minf(match, 0.35)
		completeness = minf(completeness, 0.4)
	}
	if hasFlag(in.Hallucination, string(domain.FlagContradictoryEvidence)) {
		agreement = minf(agreement, 0.35)
	}
	if hasFlag(in.Hallucination, string(domain.FlagInsufficientEvidence)) {
		quality = minf(quality, 0.3)
		count = minf(count, 0.25)
	}
	if hasFlag(in.Hallucination, string(domain.FlagUnsupportedClaim), string(domain.FlagOverconfidentConclusion), string(domain.FlagInventedRecall)) {
		reviewer = minf(reviewer, 0.4)
	}
	weakRecall := false
	for _, m := range in.Matches {
		if m.RequiresReview || m.Method == recall.MethodFuzzyName {
			weakRecall = true
		}
	}
	if weakRecall {
		match = minf(match, 0.45)
	}

	score := quality*WeightEvidenceQuality +
		count*WeightEvidenceCount +
		agreement*WeightSourceAgreement +
		freshness*WeightSourceFreshness +
		match*WeightProductMatch +
		reviewer*WeightReviewerValid +
		completeness*WeightDataCompleteness +
		coverage*WeightReviewCoverage

	return ConfidenceResult{
		Score: round2(clamp01(score)),
		Factors: map[string]float64{
			"evidenceQuality":   round2(quality),
			"evidenceCount":     round2(count),
			"sourceAgreement":   round2(agreement),
			"sourceFreshness":   round2(freshness),
			"productMatchConfidence": round2(match),
			"reviewerValidation": round2(reviewer),
			"dataCompleteness":  round2(completeness),
			"reviewCoverage":    round2(coverage),
		},
	}
}

func evidenceQuality(items []domain.Evidence) float64 {
	if len(items) == 0 {
		return 0.15
	}
	sum := 0.0
	for _, ev := range items {
		sum += ev.Reliability
	}
	return clamp01(sum / float64(len(items)))
}

func sourceFreshness(items []domain.Evidence) float64 {
	if len(items) == 0 {
		return 0.2
	}
	sum := 0.0
	for _, ev := range items {
		sum += ev.Freshness
	}
	return clamp01(sum / float64(len(items)))
}

func sourceAgreement(vals []domain.EvidenceClaimValidation, evidence []domain.Evidence) float64 {
	if len(vals) == 0 {
		if len(evidence) == 0 {
			return 0.3
		}
		return 0.6
	}
	score := 0.0
	for _, v := range vals {
		switch v.SupportStatus {
		case domain.EvidenceSupported:
			score += 1
		case domain.EvidencePartiallySupported:
			score += 0.6
		case domain.EvidenceContradicted:
			score += 0.1
		default:
			score += 0.25
		}
	}
	return clamp01(score / float64(len(vals)))
}

func productMatch(matches []domain.RecallMatchRecord) float64 {
	best := 0.5
	for _, m := range matches {
		if m.Matched && !m.RequiresReview && m.Confidence > best {
			best = m.Confidence
		}
	}
	return clamp01(best)
}

func dataCompleteness(in ConfidenceInput) float64 {
	n := 0.0
	if len(in.Evidence) > 0 {
		n += 0.34
	}
	if in.ReviewCount > 0 {
		n += 0.33
	}
	if in.HasIdentity {
		n += 0.33
	}
	return n
}

func hasFlag(flags []string, want ...string) bool {
	for _, f := range flags {
		for _, w := range want {
			if f == w {
				return true
			}
		}
	}
	return false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
