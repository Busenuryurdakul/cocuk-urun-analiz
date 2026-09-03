package compliance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConsentService struct {
	Consents   *repository.ConsentRepository
	Events     *repository.ComplianceEventRepository
	Security   *repository.SecurityEventRepository
	PolicyRepo *PolicyRepository
	Orgs       *repository.OrganizationRepository
}

func (s *ConsentService) Grant(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose, source domain.ConsentSource, policyVersion string) (*domain.Consent, error) {
	if _, err := s.Consents.FindActive(ctx, userID, orgID, purpose); err == nil {
		return nil, repository.ErrDuplicate
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	consent := &domain.Consent{
		UserID:         userID,
		OrganizationID: orgID,
		Purpose:        purpose,
		PolicyVersion:  policyVersion,
		LawfulBasis:    "consent",
		Source:         source,
	}
	if err := s.Consents.Insert(ctx, consent); err != nil {
		return nil, err
	}

	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: orgID,
		UserID:         &uid,
		EventType:      domain.EventConsentGranted,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"purpose": string(purpose), "policyVersion": policyVersion},
	})
	_ = s.Events.Record(ctx, domain.ComplianceEvent{
		OrganizationID: orgID,
		UserID:         &uid,
		EventType:      "CONSENT_GRANTED",
		Result:         "PASS",
		Details:        map[string]string{"purpose": string(purpose), "policyVersion": policyVersion},
	})
	return consent, nil
}

func (s *ConsentService) Withdraw(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose) error {
	if _, err := s.Consents.FindActive(ctx, userID, orgID, purpose); err != nil {
		return err
	}
	if err := s.Consents.Withdraw(ctx, userID, orgID, purpose); err != nil {
		return err
	}
	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: orgID,
		UserID:         &uid,
		EventType:      domain.EventConsentWithdrawn,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"purpose": string(purpose)},
	})
	return s.Events.Record(ctx, domain.ComplianceEvent{
		OrganizationID: orgID,
		UserID:         &uid,
		EventType:      "CONSENT_WITHDRAWN",
		Result:         "PASS",
		Details:        map[string]string{"purpose": string(purpose)},
	})
}

func (s *ConsentService) Current(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose) (*domain.Consent, error) {
	return s.Consents.FindActive(ctx, userID, orgID, purpose)
}

func (s *ConsentService) ListForUser(ctx context.Context, userID primitive.ObjectID) ([]domain.Consent, error) {
	return s.Consents.ListByUser(ctx, userID)
}

func (s *ConsentService) RequireConsent(ctx context.Context, userID primitive.ObjectID, orgID *primitive.ObjectID, purpose domain.ConsentPurpose) error {
	_, err := s.Consents.FindActive(ctx, userID, orgID, purpose)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w: %s", ErrConsentRequired, purpose)
	}
	return err
}

func DefaultPlatformPolicyRules(profile domain.ComplianceProfile) domain.PolicyRules {
	purposes := []string{
		string(domain.ConsentPurposeRegistration),
		string(domain.ConsentPurposeDataProcessing),
	}
	if profile == domain.ProfileGDPR || profile == domain.ProfileBoth {
		purposes = append(purposes, string(domain.ConsentPurposeOrgMembership))
	}
	return domain.PolicyRules{
		RequireConsentPurposes: purposes,
		ForbiddenClaimPatterns: []string{
			"certified safe",
			"guaranteed compliant",
			"definitely safe",
		},
	}
}

func SeedPlatformPolicies(ctx context.Context, policies *repository.CompliancePolicyRepository) error {
	for _, profile := range []domain.ComplianceProfile{domain.ProfileKVKK, domain.ProfileGDPR, domain.ProfileBoth} {
		count, err := policies.CountPlatformPublished(ctx, profile, "1.0.0")
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		now := time.Now().UTC()
		policy := &domain.CompliancePolicyVersion{
			Profile:     profile,
			Version:     "1.0.0",
			Status:      domain.PolicyStatusPublished,
			EffectiveAt: now,
			Rules:       DefaultPlatformPolicyRules(profile),
			Reason:      "platform bootstrap",
			CreatedBy:   primitive.NilObjectID,
		}
		if err := policies.Insert(ctx, policy); err != nil {
			return err
		}
	}
	return nil
}
