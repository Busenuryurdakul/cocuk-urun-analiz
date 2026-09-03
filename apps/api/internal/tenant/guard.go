package tenant

import (
	"context"
	"errors"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrCrossTenantAccess = errors.New("cross-tenant access denied")

type Guard struct {
	Members *repository.MemberRepository
	Events  *repository.SecurityEventRepository
}

func (g *Guard) RequireMembership(ctx context.Context, userID, organizationID primitive.ObjectID) (domain.OrgRole, error) {
	member, err := g.Members.FindByUserAndOrg(ctx, userID, organizationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			uid := userID
			oid := organizationID
			_ = g.Events.Record(ctx, domain.SecurityEvent{
				OrganizationID: &oid,
				UserID:         &uid,
				EventType:      domain.EventCrossTenantAccess,
				Severity:       domain.SeverityCritical,
				Details:        map[string]string{"reason": "membership_not_found"},
			})
			return "", ErrCrossTenantAccess
		}
		return "", err
	}
	return member.Role, nil
}
