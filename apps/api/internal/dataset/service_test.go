package dataset_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestEvaluateEligibilityTrainingApproved(t *testing.T) {
	got := dataset.EvaluateEligibility(dataset.EligibilityInput{
		ModerationStatus:  domain.ModerationApproved,
		QualityStatus:     domain.QualityApproved,
		PIIStatus:         domain.PIIClean,
		ProvenanceStatus:  domain.ProvenanceVerified,
		LicenseStatus:     domain.LicenseApproved,
		UsageRightsStatus: domain.UsageRightsApproved,
		SpamStatus:        domain.SpamClean,
		DuplicateStatus:   domain.DuplicateUnique,
	})
	if got != domain.EligibilityTrainingApproved {
		t.Fatalf("expected TRAINING_APPROVED, got %s", got)
	}
}

func TestEvaluateEligibilityRejected(t *testing.T) {
	got := dataset.EvaluateEligibility(dataset.EligibilityInput{
		ModerationStatus: domain.ModerationRejected,
	})
	if got != domain.EligibilityRejected {
		t.Fatalf("expected REJECTED, got %s", got)
	}
}

func TestQuarantinedEligibilityFromPartialProvenance(t *testing.T) {
	got := dataset.EvaluateEligibility(dataset.EligibilityInput{
		ModerationStatus:  domain.ModerationApproved,
		QualityStatus:     domain.QualityApproved,
		PIIStatus:         domain.PIIClean,
		ProvenanceStatus:  domain.ProvenancePartial,
		LicenseStatus:     domain.LicenseUnknown,
		UsageRightsStatus: domain.UsageRightsUnknown,
		SpamStatus:        domain.SpamClean,
		DuplicateStatus:   domain.DuplicateUnique,
	})
	if got == domain.EligibilityTrainingApproved {
		t.Fatalf("partial provenance must not yield TRAINING_APPROVED, got %s", got)
	}
}

func TestHeldOutIsolation(t *testing.T) {
	split := domain.SplitHeldOutEval
	record := domain.DatasetRecord{Split: &split}
	if !dataset.IsHeldOut(record) {
		t.Fatal("expected held-out record")
	}
	train := domain.SplitTrain
	trainRecord := domain.DatasetRecord{Split: &train}
	if dataset.IsHeldOut(trainRecord) {
		t.Fatal("train split must not be held-out")
	}
}
