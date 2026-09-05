package rbac_test

import (
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
)

func TestPhase4RBAC(t *testing.T) {
	if !rbac.CanManageProducts(domain.RoleAnalyst) {
		t.Fatal("analyst should manage products")
	}
	if rbac.CanManageDataset(domain.RoleAnalyst) {
		t.Fatal("analyst must not manage dataset")
	}
	if !rbac.CanSubmitUGC(domain.RoleViewer) {
		t.Fatal("viewer should submit ugc")
	}
}
