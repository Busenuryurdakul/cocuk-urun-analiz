package analysis

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func TestHallucinationGuard(t *testing.T) {
	invented := ApplyHallucinationGuard(GuardInput{
		WorkerText: "This product has an official CPSC recall.",
		Matches:    []domain.RecallMatchRecord{{Matched: true, RequiresReview: true, Method: recall.MethodFuzzyName, Confidence: 0.4}},
	})
	if !hasFlag(invented.Flags, string(domain.FlagInventedRecall)) {
		t.Fatalf("expected invented recall: %v", invented.Flags)
	}
	for _, m := range invented.Matches {
		if m.RequiresReview {
			t.Fatal("invented/weak recall must be removed from confirmed list")
		}
	}

	unsupported := ApplyHallucinationGuard(GuardInput{
		Validations: []domain.EvidenceClaimValidation{{SupportStatus: domain.EvidenceUnsupported}},
		WorkerText:  "definitely safe",
	})
	if !hasFlag(unsupported.Flags, string(domain.FlagUnsupportedClaim)) {
		t.Fatal("unsupported claim")
	}

	contradiction := ApplyHallucinationGuard(GuardInput{
		Validations: []domain.EvidenceClaimValidation{{SupportStatus: domain.EvidenceContradicted}},
	})
	if !hasFlag(contradiction.Flags, string(domain.FlagContradictoryEvidence)) {
		t.Fatal("contradiction")
	}

	mismatch := ApplyHallucinationGuard(GuardInput{
		WorkerText: "According to CPSC this is recalled.",
		Evidence:   []domain.Evidence{{Source: "blog", SourceType: "URL"}},
	})
	if !hasFlag(mismatch.Flags, string(domain.FlagSourceMismatch), string(domain.FlagInventedRecall)) {
		t.Fatalf("source mismatch or invented recall expected: %v", mismatch.Flags)
	}

	over := ApplyHallucinationGuard(GuardInput{WorkerText: "This is definitely safe", Evidence: nil})
	if !hasFlag(over.Flags, string(domain.FlagOverconfidentConclusion), string(domain.FlagInsufficientEvidence)) {
		t.Fatalf("overconfidence expected: %v", over.Flags)
	}
}
