package analysis

import (
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

type GuardInput struct {
	WorkerText     string
	ReviewerText   string
	Evidence       []domain.Evidence
	Validations    []domain.EvidenceClaimValidation
	Findings       []domain.SafetyFinding
	Matches        []domain.RecallMatchRecord
	ReviewCoverage int
}

type GuardResult struct {
	Flags       []string
	Limitations []string
	Matches     []domain.RecallMatchRecord
}

func ApplyHallucinationGuard(in GuardInput) GuardResult {
	out := GuardResult{Flags: []string{}, Limitations: []string{}, Matches: append([]domain.RecallMatchRecord{}, in.Matches...)}
	blob := strings.ToLower(in.WorkerText + "\n" + in.ReviewerText)

	confirmedRecall := false
	kept := make([]domain.RecallMatchRecord, 0, len(out.Matches))
	for _, m := range out.Matches {
		if m.Matched && !m.RequiresReview && m.Confidence >= recall.ConfirmedMatchMinConfidence {
			confirmedRecall = true
			kept = append(kept, m)
			continue
		}
		if m.RequiresReview || m.Method == recall.MethodFuzzyName {
			out.Limitations = appendUnique(out.Limitations, "Weak or fuzzy recall match was not treated as confirmed.")
			continue
		}
		if m.Matched {
			kept = append(kept, m)
		}
	}
	out.Matches = kept

	if mentionsRecall(blob) && !confirmedRecall && !hasOfficialRecallEvidence(in.Evidence) {
		out.Flags = appendUnique(out.Flags, string(domain.FlagInventedRecall))
		out.Limitations = appendUnique(out.Limitations, "Model mentioned a recall that is not backed by an official confirmed match.")
	}

	unsupportedCritical := false
	contradicted := false
	for _, v := range in.Validations {
		switch v.SupportStatus {
		case domain.EvidenceUnsupported:
			unsupportedCritical = true
		case domain.EvidenceContradicted:
			contradicted = true
		}
	}
	if unsupportedCritical {
		out.Flags = appendUnique(out.Flags, string(domain.FlagUnsupportedClaim))
		out.Limitations = appendUnique(out.Limitations, "Unsupported claims were downgraded and are not shown as verified facts.")
	}
	if contradicted {
		out.Flags = appendUnique(out.Flags, string(domain.FlagContradictoryEvidence))
		out.Limitations = appendUnique(out.Limitations, "Contradictory evidence requires human review.")
	}
	if len(in.Evidence) == 0 && len(in.Findings) == 0 {
		out.Flags = appendUnique(out.Flags, string(domain.FlagInsufficientEvidence))
		out.Limitations = appendUnique(out.Limitations, "No supporting evidence was persisted for this analysis.")
	}
	if overconfident(blob) && (unsupportedCritical || len(in.Evidence) == 0) {
		out.Flags = appendUnique(out.Flags, string(domain.FlagOverconfidentConclusion))
		out.Limitations = appendUnique(out.Limitations, "Overconfident language was stripped from the user-facing conclusion.")
	}
	if sourceMismatch(blob, in.Evidence, in.Matches) {
		out.Flags = appendUnique(out.Flags, string(domain.FlagSourceMismatch))
		out.Limitations = appendUnique(out.Limitations, "A cited source does not match persisted evidence.")
	}
	return out
}

func mentionsRecall(blob string) bool {
	return strings.Contains(blob, "recall") || strings.Contains(blob, "geri çağır") || strings.Contains(blob, "toplatma")
}

func hasOfficialRecallEvidence(items []domain.Evidence) bool {
	for _, ev := range items {
		if ev.SourceType == "OFFICIAL_RECALL" || ev.SourceType == "OFFICIAL_SAFETY_NOTICE" {
			return true
		}
	}
	return false
}

func overconfident(blob string) bool {
	for _, marker := range strings.Split(OverconfidenceMarkers, "|") {
		if strings.Contains(blob, marker) {
			return true
		}
	}
	return false
}

func sourceMismatch(blob string, evidence []domain.Evidence, matches []domain.RecallMatchRecord) bool {
	if !strings.Contains(blob, "cpsc") && !strings.Contains(blob, "gübi") && !strings.Contains(blob, "gubis") {
		return false
	}
	for _, ev := range evidence {
		src := strings.ToLower(ev.Source + " " + ev.SourceType)
		if strings.Contains(src, "cpsc") || strings.Contains(src, "gubis") {
			return false
		}
	}
	for _, m := range matches {
		if strings.EqualFold(m.Source, "CPSC") || strings.EqualFold(m.Source, "GUBIS") {
			return false
		}
	}
	return true
}

func appendUnique(items []string, v string) []string {
	for _, existing := range items {
		if existing == v {
			return items
		}
	}
	return append(items, v)
}
