package rbac

import "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"

func CanStartAnalysisRun(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst:
		return true
	default:
		return false
	}
}

func CanReadAnalysisRun(role domain.OrgRole) bool {
	return CanReadProducts(role)
}

func CanCancelAnalysisRun(role domain.OrgRole, isOwner bool) bool {
	if role == domain.RoleOwner || role == domain.RoleAdmin {
		return true
	}
	return isOwner && (role == domain.RoleAnalyst)
}
