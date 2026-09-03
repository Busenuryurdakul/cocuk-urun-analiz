package compliance

import (
	"context"
	"errors"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidPolicyVersion = errors.New("invalid compliance policy version pin")

type PolicyRepository struct {
	Policies *repository.CompliancePolicyRepository
}

// ResolveActive returns the concrete published policy for an organization.
// Precedence: org-specific published override for pinned version, then platform published default.
// Runtime "latest" resolution is forbidden; organizations must pin a specific published version.
func (p *PolicyRepository) ResolveActive(ctx context.Context, org *domain.Organization) (*domain.CompliancePolicyVersion, error) {
	profile := domain.ComplianceProfile(org.ComplianceProfile)
	if !ProfileAlwaysOn(profile) {
		return nil, ErrInvalidProfile
	}

	versionPin := strings.TrimSpace(org.CompliancePolicyVersion)
	if versionPin == "" || strings.EqualFold(versionPin, "latest") {
		return nil, ErrInvalidPolicyVersion
	}

	orgID := org.ID
	policy, err := p.Policies.FindPublished(ctx, &orgID, profile, versionPin)
	if err == nil {
		return policy, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return p.Policies.FindPublished(ctx, nil, profile, versionPin)
}

// ResolvePlatformDefault returns the platform bootstrap published policy for a profile.
func (p *PolicyRepository) ResolvePlatformDefault(ctx context.Context, profile domain.ComplianceProfile) (*domain.CompliancePolicyVersion, error) {
	return p.Policies.FindPublished(ctx, nil, profile, PlatformDefaultPolicyVersion)
}

func (p *PolicyRepository) PublishOrgPolicy(ctx context.Context, orgID, actorID primitive.ObjectID, profile domain.ComplianceProfile, version, reason string, rules domain.PolicyRules) (*domain.CompliancePolicyVersion, error) {
	if err := p.Policies.ArchiveOrgPublished(ctx, orgID, profile); err != nil {
		return nil, err
	}

	policy := &domain.CompliancePolicyVersion{
		OrganizationID: &orgID,
		Profile:        profile,
		Version:        version,
		Status:         domain.PolicyStatusPublished,
		Rules:          rules,
		Reason:         reason,
		CreatedBy:      actorID,
	}
	if err := p.Policies.Insert(ctx, policy); err != nil {
		return nil, err
	}
	return policy, nil
}
