package rbac_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
)

func TestRBACMatrix(t *testing.T) {
	if !rbac.CanManageMembers(domain.RoleOwner) || !rbac.CanManageMembers(domain.RoleAdmin) {
		t.Fatal("owner/admin should manage members")
	}
	if rbac.CanManageMembers(domain.RoleViewer) {
		t.Fatal("viewer must not manage members")
	}
	if !rbac.CanUpdateCompliance(domain.RoleAdmin) {
		t.Fatal("admin should update compliance")
	}
	if rbac.CanPublishPolicy(domain.RoleAnalyst) {
		t.Fatal("analyst must not publish policy")
	}
}
