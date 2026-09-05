package marketplace_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
)

func TestAmazonBabyGovernanceNotTrainingEligible(t *testing.T) {
	g := importpkg.AmazonBabyDefaultGovernance()
	if dataset.IsTrainingEligible(g.DatasetEligibility) {
		t.Fatal("amazon baby must not be training eligible by default")
	}
	if !dataset.RejectsTrainingSelection(g.DatasetEligibility) {
		t.Fatal("quarantined records must reject training selection")
	}
}

func TestQuarantinedEligibilityRejectedForTraining(t *testing.T) {
	cases := []domain.DatasetEligibility{
		domain.EligibilityQuarantined,
		domain.EligibilityEvalOnly,
		domain.EligibilityAnalysisOnly,
		domain.EligibilityRejected,
	}
	for _, c := range cases {
		if dataset.IsTrainingEligible(c) {
			t.Fatalf("%s should not be training eligible", c)
		}
	}
}

func TestTrainingApprovedOnlyForExplicitApproval(t *testing.T) {
	if !dataset.IsTrainingEligible(domain.EligibilityTrainingApproved) {
		t.Fatal("TRAINING_APPROVED must remain eligible")
	}
	if dataset.RejectsTrainingSelection(domain.EligibilityTrainingApproved) {
		t.Fatal("TRAINING_APPROVED must not reject training selection")
	}
}
