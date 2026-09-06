package rbac

import "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"

func CanReadLLM(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleViewer:
		return true
	default:
		return false
	}
}

func CanConfigureLLM(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}

func CanPublishLLM(role domain.OrgRole) bool {
	return CanConfigureLLM(role)
}

func CanRollbackLLM(role domain.OrgRole) bool {
	return role == domain.RoleOwner
}

func CanTestLLM(role domain.OrgRole) bool {
	return CanConfigureLLM(role)
}

func CanReadLLMUsage(role domain.OrgRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst:
		return true
	default:
		return false
	}
}

func CanReadLLMAudit(role domain.OrgRole) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}
