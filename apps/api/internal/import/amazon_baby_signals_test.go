package importpkg_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
)

func TestDeriveSentimentFromRating(t *testing.T) {
	r1 := 1.0
	s, src := importpkg.DeriveSentimentFromRating(&r1)
	if s != importpkg.SentimentNegative || src != importpkg.LabelSourceRatingDerived {
		t.Fatalf("unexpected sentiment for rating 1: %s", s)
	}
	r3 := 3.0
	s, _ = importpkg.DeriveSentimentFromRating(&r3)
	if s != importpkg.SentimentNeutral {
		t.Fatalf("expected neutral for rating 3")
	}
	r5 := 5.0
	s, _ = importpkg.DeriveSentimentFromRating(&r5)
	if s != importpkg.SentimentPositive {
		t.Fatalf("expected positive for rating 5")
	}
}

func TestDeriveIssueType(t *testing.T) {
	issue, method, conf := importpkg.DeriveIssueType("The product broke during first use")
	if issue != domain.IssueBreakage {
		t.Fatalf("expected BREAKAGE, got %s", issue)
	}
	if method != importpkg.LabelMethodRuleBased {
		t.Fatalf("expected rule based method")
	}
	if conf == "" {
		t.Fatal("expected confidence")
	}
}

func TestDeriveSafetyObservation(t *testing.T) {
	v := importpkg.DeriveSafetyObservation("There is a choking hazard with small parts")
	if v == nil || !*v {
		t.Fatal("expected safety-related observation true")
	}
	v2 := importpkg.DeriveSafetyObservation("Soft and comfortable blanket")
	if v2 == nil || *v2 {
		t.Fatal("expected safety-related observation false")
	}
}

func TestAmazonBabyDefaultGovernance(t *testing.T) {
	g := importpkg.AmazonBabyDefaultGovernance()
	if g.ProvenanceStatus != domain.ProvenancePartial ||
		g.LicenseStatus != domain.LicenseUnknown ||
		g.UsageRightsStatus != domain.UsageRightsUnknown ||
		g.DatasetEligibility != domain.EligibilityQuarantined ||
		g.TrainingAllowed || g.EvaluationAllowed {
		t.Fatalf("unexpected governance defaults: %+v", g)
	}
}
