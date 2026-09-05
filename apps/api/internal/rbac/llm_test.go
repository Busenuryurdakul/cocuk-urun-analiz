package rbac

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestLLMPermissions(t *testing.T) {
	if !CanReadLLM(domain.RoleViewer) {
		t.Fatal("viewer should read llm")
	}
	if CanConfigureLLM(domain.RoleAnalyst) {
		t.Fatal("analyst should not configure llm")
	}
	if !CanPublishLLM(domain.RoleAdmin) {
		t.Fatal("admin should publish llm")
	}
	if CanRollbackLLM(domain.RoleAdmin) {
		t.Fatal("rollback owner only")
	}
	if !CanRollbackLLM(domain.RoleOwner) {
		t.Fatal("owner should rollback")
	}
}
