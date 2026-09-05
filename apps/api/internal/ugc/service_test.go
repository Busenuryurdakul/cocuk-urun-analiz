package ugc_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/ugc"
)

func TestUGCValidationRejectsExtraFields(t *testing.T) {
	err := compliance.ValidateMinimization("create_user_experience", map[string]string{
		"narrative":         "works well",
		"usageStatus":       string(domain.UsageUsed),
		"satisfactionLevel": string(domain.SatisfactionSatisfied),
		"email":             "secret@example.com",
	})
	if err == nil {
		t.Fatal("expected minimization violation")
	}
}

func TestUGCValidationRequiresNarrative(t *testing.T) {
	err := ugc.ValidateCreateInput(ugc.CreateInput{
		UsageStatus:       domain.UsageUsed,
		SatisfactionLevel: domain.SatisfactionSatisfied,
		Narrative:         "   ",
	})
	if err == nil {
		t.Fatal("expected invalid narrative")
	}
}
