package analysis

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDecisionMatrix(t *testing.T) {
	evID := primitive.NewObjectID()
	cases := []struct {
		name string
		in   DecisionInput
		want domain.ProductDecision
	}{
		{
			name: "critical confirmed recall",
			in: DecisionInput{Findings: []domain.SafetyFinding{{
				Type: domain.SafetyRecall, Severity: domain.SafetySeverityCritical, Confidence: 0.98, EvidenceIDs: []primitive.ObjectID{evID},
			}}},
			want: domain.DecisionBlock,
		},
		{
			name: "weak recall match",
			in: DecisionInput{Matches: []domain.RecallMatchRecord{{
				Matched: true, RequiresReview: true, Method: recall.MethodFuzzyName, Confidence: 0.42,
			}}},
			want: domain.DecisionReviewRequired,
		},
		{
			name: "medium supported warning",
			in: DecisionInput{Findings: []domain.SafetyFinding{{
				Type: domain.SafetyHygiene, Severity: domain.SafetySeverityMedium, EvidenceIDs: []primitive.ObjectID{evID},
			}}},
			want: domain.DecisionAllowWithWarning,
		},
		{
			name: "no supported risk",
			in:   DecisionInput{},
			want: domain.DecisionAllow,
		},
		{
			name: "contradictory evidence",
			in: DecisionInput{Validations: []domain.EvidenceClaimValidation{{
				SupportStatus: domain.EvidenceContradicted, ClaimText: "safe",
			}}},
			want: domain.DecisionReviewRequired,
		},
		{
			name: "unsupported claim does not block",
			in:   DecisionInput{Flags: []string{string(domain.FlagUnsupportedClaim)}},
			want: domain.DecisionReviewRequired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.in)
			if got.Decision != tc.want {
				t.Fatalf("got %s want %s (%s)", got.Decision, tc.want, got.Recommendation)
			}
		})
	}
}

func TestFalseBlockGuard(t *testing.T) {
	weak := Decide(DecisionInput{Matches: []domain.RecallMatchRecord{{
		Matched: true, RequiresReview: true, Method: recall.MethodFuzzyName, Confidence: 0.42,
	}}})
	if weak.Decision == domain.DecisionBlock {
		t.Fatal("weak fuzzy recall must not BLOCK")
	}
	unsupported := Decide(DecisionInput{Flags: []string{string(domain.FlagUnsupportedClaim)}})
	if unsupported.Decision == domain.DecisionBlock {
		t.Fatal("unsupported claim must not BLOCK")
	}
}
