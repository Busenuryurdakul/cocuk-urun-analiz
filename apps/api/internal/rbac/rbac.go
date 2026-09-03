package rbac

import "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"

func CanReadWorkspace(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleViewer:
		return true
	default:
		return false
	}
}

func CanManageMembers(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}
