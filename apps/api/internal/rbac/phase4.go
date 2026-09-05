package rbac

import "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"

func CanReadProducts(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleViewer:
		return true
	default:
		return false
	}
}

func CanManageProducts(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin || role == domain.RoleAnalyst
}

func CanReadUGC(role domain.OrgRole) bool {
	return CanReadProducts(role)
}

func CanManageUGC(role domain.OrgRole) bool {
	return CanManageProducts(role)
}

func CanSubmitUGC(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleViewer:
		return true
	default:
		return false
	}
}

func CanModerateUGC(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}

func CanManageMarketplaceImport(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin || role == domain.RoleAnalyst
}

func CanReadMarketplaceImport(role domain.OrgRole) bool {
	return CanReadProducts(role)
}

func CanReadDataset(role domain.OrgRole) bool {
	return CanReadProducts(role)
}

func CanManageDataset(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}
