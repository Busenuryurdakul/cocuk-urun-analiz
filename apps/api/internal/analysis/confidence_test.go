package analysis

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func TestConfidenceOrdering(t *testing.T) {
	strong := ComputeConfidence(ConfidenceInput{
		Evidence: []domain.Evidence{
			{Reliability: 0.95, Freshness: 0.9, SourceType: "OFFICIAL_RECALL"},
			{Reliability: 0.9, Freshness: 0.8, SourceType: "URL"},
			{Reliability: 0.88, Freshness: 0.8, SourceType: "MANUAL"},
		},
		Validations:    []domain.EvidenceClaimValidation{{SupportStatus: domain.EvidenceSupported}},
		Matches:        []domain.RecallMatchRecord{{Matched: true, RequiresReview: false, Confidence: 0.97}},
		ReviewCount:    12,
		HasIdentity:    true,
		ReviewerPassed: true,
	})
	weak := ComputeConfidence(ConfidenceInput{
		Evidence:    []domain.Evidence{{Reliability: 0.3, Freshness: 0.2}},
		ReviewCount: 1,
		HasIdentity: false,
		Hallucination: []string{string(domain.FlagInsufficientEvidence)},
	})
	conflict := ComputeConfidence(ConfidenceInput{
		Evidence: []domain.Evidence{{Reliability: 0.7, Freshness: 0.7}, {Reliability: 0.7, Freshness: 0.6}},
		Validations: []domain.EvidenceClaimValidation{{SupportStatus: domain.EvidenceContradicted}},
		Hallucination: []string{string(domain.FlagContradictoryEvidence)},
		HasIdentity: true,
		ReviewCount: 4,
	})
	if strong.Score <= weak.Score {
		t.Fatalf("strong=%v weak=%v", strong.Score, weak.Score)
	}
	if conflict.Score >= strong.Score {
		t.Fatalf("conflict should be lower than strong: %v vs %v", conflict.Score, strong.Score)
	}
	flagged := ComputeConfidence(ConfidenceInput{
		Evidence: []domain.Evidence{{Reliability: 0.9, Freshness: 0.9}},
		HasIdentity: true, ReviewCount: 8, ReviewerPassed: true,
		Hallucination: []string{string(domain.FlagInventedRecall)},
	})
	clean := ComputeConfidence(ConfidenceInput{
		Evidence: []domain.Evidence{{Reliability: 0.9, Freshness: 0.9}},
		HasIdentity: true, ReviewCount: 8, ReviewerPassed: true,
	})
	if flagged.Score >= clean.Score {
		t.Fatalf("hallucination flags must lower confidence: %v vs %v", flagged.Score, clean.Score)
	}
	_ = recall.ConfirmedMatchMinConfidence
}
