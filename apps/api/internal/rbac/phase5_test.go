package rbac

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestCanStartAnalysisRun(t *testing.T) {
	if !CanStartAnalysisRun(domain.RoleAnalyst) {
		t.Fatal("analyst should start analysis")
	}
	if CanStartAnalysisRun(domain.RoleViewer) {
		t.Fatal("viewer should not start analysis")
	}
}

func TestCanCancelAnalysisRun(t *testing.T) {
	if !CanCancelAnalysisRun(domain.RoleAdmin, false) {
		t.Fatal("admin should cancel org run")
	}
	if !CanCancelAnalysisRun(domain.RoleAnalyst, true) {
		t.Fatal("analyst should cancel own run")
	}
	if CanCancelAnalysisRun(domain.RoleAnalyst, false) {
		t.Fatal("analyst should not cancel others run")
	}
}
