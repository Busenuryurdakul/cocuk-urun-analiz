package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden          = errors.New("forbidden")
	ErrLastOwner          = errors.New("last owner protection")
	ErrPersonalOrg        = errors.New("personal organization immutable")
	ErrInvalidRole        = errors.New("invalid role")
	ErrAlreadyMember      = errors.New("already a member")
	ErrInvitationPending  = errors.New("invitation already pending")
	ErrInvitationExpired  = errors.New("invitation expired")
	ErrInvitationMismatch = errors.New("invitation email mismatch")
	ErrInvitationUsed     = errors.New("invitation already used")
)

const invitationTTL = 7 * 24 * time.Hour

type Service struct {
	Orgs        *repository.OrganizationRepository
	Members     *repository.MemberRepository
	Users       *repository.UserRepository
	Invitations *repository.InvitationRepository
	Security    *repository.SecurityEventRepository
	ConfigAudit *repository.ConfigAuditRepository
	Policies    *repository.CompliancePolicyRepository
	Mail        mail.Service
	Tenant      *tenant.Guard
	Compliance  *compliance.Engine
	Consent     *compliance.ConsentService
	PolicyRepo  *compliance.PolicyRepository
	WebBaseURL  string
}

type MemberView struct {
	UserID    primitive.ObjectID
	Email     string
	Role      domain.OrgRole
	JoinedAt  time.Time
	InvitedAt time.Time
}

func (s *Service) CreateOrganization(ctx context.Context, actorID primitive.ObjectID, name, profileRaw string) (*domain.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required", compliance.ErrComplianceViolation)
	}

	profile, err := compliance.ValidateProfile(profileRaw)
	if err != nil {
		return nil, err
	}

	policy, err := s.PolicyRepo.ResolvePlatformDefault(ctx, profile)
	if err != nil {
		return nil, err
	}

	user, err := s.Users.FindByID(ctx, actorID)
	if err != nil {
		return nil, err
	}

	fields := map[string]string{"name": name, "complianceProfile": string(profile)}
	if _, err := s.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "create_organization",
		OrganizationID: user.PersonalOrgID,
		UserID:         actorID,
		ProfileInput:   string(profile),
		Fields:         fields,
	}); err != nil {
		if errors.Is(err, compliance.ErrBypassAttempt) || errors.Is(err, compliance.ErrInvalidProfile) || errors.Is(err, compliance.ErrComplianceViolation) {
			return nil, err
		}
	}

	org := &domain.Organization{
		Name:                    name,
		Type:                    domain.OrgTypeOrganization,
		ComplianceProfile:       string(profile),
		CompliancePolicyVersion: policy.Version,
		OwnerID:                 actorID,
	}
	if err := s.Orgs.Create(ctx, org); err != nil {
		return nil, err
	}

	member := &domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         actorID,
		Role:           domain.RoleOwner,
	}
	if err := s.Members.Create(ctx, member); err != nil {
		return nil, err
	}

	uid := actorID
	oid := org.ID
	s.grantRequiredConsents(ctx, actorID, oid, profile, policy.Version)
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventOrgCreated,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"name": compliance.SanitizeForAudit(name), "profile": string(profile)},
	})
	return org, nil
}

func (s *Service) grantRequiredConsents(ctx context.Context, userID, orgID primitive.ObjectID, profile domain.ComplianceProfile, policyVersion string) {
	if s.Consent == nil {
		return
	}
	orgPtr := &orgID
	for _, purposeRaw := range compliance.DefaultPlatformPolicyRules(profile).RequireConsentPurposes {
		_, _ = s.Consent.Grant(ctx, userID, orgPtr, domain.ConsentPurpose(purposeRaw), domain.ConsentSourceWeb, policyVersion)
	}
}

func (s *Service) InviteMember(ctx context.Context, actorID, orgID primitive.ObjectID, email, roleRaw string) error {
	role, err := parseAssignableRole(roleRaw)
	if err != nil {
		return err
	}

	actorRole, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return err
	}
	if !rbac.CanManageMembers(actorRole) {
		return ErrForbidden
	}

	org, err := s.Orgs.FindByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org.Type == domain.OrgTypePersonal {
		return ErrPersonalOrg
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return fmt.Errorf("%w: email required", compliance.ErrComplianceViolation)
	}

	if existingUser, userErr := s.Users.FindByEmail(ctx, email); userErr == nil {
		if _, memErr := s.Members.FindByUserAndOrg(ctx, existingUser.ID, orgID); memErr == nil {
			return ErrAlreadyMember
		} else if memErr != nil && !errors.Is(memErr, repository.ErrNotFound) {
			return memErr
		}
	} else if !errors.Is(userErr, repository.ErrNotFound) {
		return userErr
	}

	s.grantRequiredConsents(ctx, actorID, orgID, domain.ComplianceProfile(org.ComplianceProfile), org.CompliancePolicyVersion)

	fields := map[string]string{"email": email, "role": string(role)}
	if _, err := s.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "invite_member",
		OrganizationID: orgID,
		UserID:         actorID,
		Fields:         fields,
	}); err != nil {
		return err
	}

	token, tokenHash, err := auth.NewToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(invitationTTL)

	if pending, invErr := s.Invitations.FindPendingByOrgAndEmail(ctx, orgID, email); invErr == nil {
		if err := s.Invitations.RefreshPending(ctx, pending.ID, tokenHash, expiresAt); err != nil {
			return err
		}
	} else if errors.Is(invErr, repository.ErrNotFound) {
		inv := &domain.OrganizationInvitation{
			OrganizationID: orgID,
			Email:          email,
			Role:           role,
			InviterID:      actorID,
			TokenHash:      tokenHash,
			Status:         domain.InvitationPending,
			ExpiresAt:      expiresAt,
		}
		if err := s.Invitations.Create(ctx, inv); err != nil {
			return err
		}
	} else {
		return invErr
	}

	if err := s.sendInvitationMail(ctx, email, org.Name, token); err != nil {
		return err
	}

	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &orgID,
		UserID:         &uid,
		EventType:      domain.EventOrgMemberInvited,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"email": compliance.SanitizeForAudit(email), "role": string(role)},
	})
	return nil
}

func (s *Service) sendInvitationMail(ctx context.Context, email, orgName, token string) error {
	acceptURL := fmt.Sprintf("%s/org/accept-invite?token=%s", strings.TrimRight(s.WebBaseURL, "/"), token)
	subject, plain, html := mail.InvitationEmail(orgName, acceptURL)
	if err := mail.DeliverNow(ctx, s.Mail, mail.Message{
		To: email, Subject: subject, Body: plain, HTMLBody: html,
	}); err != nil {
		return fmt.Errorf("invitation mail: %w", err)
	}
	return nil
}

func (s *Service) AcceptInvitation(ctx context.Context, actorID primitive.ObjectID, token string) (*domain.Organization, error) {
	hash := auth.HashToken(token)
	inv, err := s.Invitations.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			uid := actorID
			_ = s.Security.Record(ctx, domain.SecurityEvent{
				UserID:    &uid,
				EventType: domain.EventInvitationReplay,
				Severity:  domain.SeverityWarning,
				Details:   map[string]string{"reason": "invalid_token"},
			})
		}
		return nil, ErrInvitationUsed
	}

	if inv.Status != domain.InvitationPending {
		uid := actorID
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			OrganizationID: &inv.OrganizationID,
			UserID:         &uid,
			EventType:      domain.EventInvitationReplay,
			Severity:       domain.SeverityWarning,
			Details:        map[string]string{"reason": "not_pending"},
		})
		return nil, ErrInvitationUsed
	}

	if time.Now().UTC().After(inv.ExpiresAt) {
		_ = s.Invitations.MarkExpired(ctx, inv.ID)
		return nil, ErrInvitationExpired
	}

	user, err := s.Users.FindByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(user.Email) != inv.Email {
		return nil, ErrInvitationMismatch
	}

	if _, err := s.Members.FindByUserAndOrg(ctx, actorID, inv.OrganizationID); err == nil {
		return nil, repository.ErrDuplicate
	}

	member := &domain.OrganizationMember{
		OrganizationID: inv.OrganizationID,
		UserID:         actorID,
		Role:           inv.Role,
	}
	if err := s.Members.Create(ctx, member); err != nil {
		return nil, err
	}
	if err := s.Invitations.MarkAccepted(ctx, inv.ID, actorID); err != nil {
		return nil, err
	}

	org, err := s.Orgs.FindByID(ctx, inv.OrganizationID)
	if err != nil {
		return nil, err
	}

	oid := inv.OrganizationID
	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventOrgMemberJoined,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"role": string(inv.Role)},
	})

	policy, _ := s.PolicyRepo.ResolveActive(ctx, org)
	policyVersion := "1.0.0"
	profile := domain.ComplianceProfile(org.ComplianceProfile)
	if policy != nil {
		policyVersion = policy.Version
		profile = policy.Profile
	}
	s.grantRequiredConsents(ctx, actorID, oid, profile, policyVersion)

	return org, nil
}

func (s *Service) ListMembers(ctx context.Context, actorID, orgID primitive.ObjectID) ([]MemberView, error) {
	if _, err := s.Tenant.RequireMembership(ctx, actorID, orgID); err != nil {
		return nil, err
	}
	members, err := s.Members.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberView, 0, len(members))
	for _, m := range members {
		user, err := s.Users.FindByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		out = append(out, MemberView{
			UserID:    m.UserID,
			Email:     user.Email,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
			InvitedAt: m.InvitedAt,
		})
	}
	return out, nil
}

func (s *Service) UpdateMemberRole(ctx context.Context, actorID, orgID, targetUserID primitive.ObjectID, roleRaw string) error {
	newRole, err := parseAssignableRole(roleRaw)
	if err != nil {
		return err
	}

	actorRole, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return err
	}
	if !rbac.CanManageMembers(actorRole) {
		return ErrForbidden
	}

	target, err := s.Members.FindByUserAndOrg(ctx, targetUserID, orgID)
	if err != nil {
		return err
	}

	if target.Role == domain.RoleOwner && newRole != domain.RoleOwner {
		count, err := s.Members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	if err := s.Members.UpdateRole(ctx, orgID, targetUserID, newRole); err != nil {
		return err
	}

	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &orgID,
		UserID:         &uid,
		EventType:      domain.EventOrgRoleChanged,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"targetUserId": targetUserID.Hex(), "role": string(newRole)},
	})
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, actorID, orgID, targetUserID primitive.ObjectID) error {
	actorRole, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return err
	}
	if !rbac.CanManageMembers(actorRole) {
		return ErrForbidden
	}

	target, err := s.Members.FindByUserAndOrg(ctx, targetUserID, orgID)
	if err != nil {
		return err
	}

	if target.Role == domain.RoleOwner {
		count, err := s.Members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastOwner
		}
	}

	if err := s.Members.Delete(ctx, orgID, targetUserID); err != nil {
		return err
	}

	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &orgID,
		UserID:         &uid,
		EventType:      domain.EventOrgMemberRemoved,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"targetUserId": targetUserID.Hex()},
	})
	return nil
}

func (s *Service) UpdateComplianceProfile(ctx context.Context, actorID, orgID primitive.ObjectID, profileRaw string) (*domain.Organization, error) {
	actorRole, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanUpdateCompliance(actorRole) {
		return nil, ErrForbidden
	}

	org, err := s.Orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org.Type == domain.OrgTypePersonal {
		return nil, ErrPersonalOrg
	}

	profile, err := compliance.ValidateProfile(profileRaw)
	if err != nil {
		return nil, err
	}

	fields := map[string]string{"complianceProfile": string(profile)}
	if _, err := s.Compliance.Evaluate(ctx, compliance.EvaluateRequest{
		Operation:      "update_compliance_profile",
		OrganizationID: orgID,
		UserID:         actorID,
		ProfileInput:   string(profile),
		Fields:         fields,
	}); err != nil {
		return nil, err
	}

	policy, err := s.PolicyRepo.ResolvePlatformDefault(ctx, profile)
	if err != nil {
		return nil, err
	}

	oldProfile := org.ComplianceProfile
	if err := s.Orgs.UpdateCompliance(ctx, orgID, string(profile), policy.Version); err != nil {
		return nil, err
	}

	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &orgID,
		UserID:         &uid,
		EventType:      domain.EventComplianceProfileChange,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"oldProfile": oldProfile, "newProfile": string(profile), "policyVersion": policy.Version},
	})
	_ = s.ConfigAudit.Record(ctx, domain.ConfigAuditLog{
		ChangedBy:  actorID,
		Reason:     "compliance profile update",
		OldVersion: org.CompliancePolicyVersion,
		NewVersion: policy.Version,
		FieldDiff:  map[string]string{"complianceProfile": oldProfile + "->" + string(profile)},
		OrgID:      &orgID,
	})

	return s.Orgs.FindByID(ctx, orgID)
}

func (s *Service) PublishCompliancePolicy(ctx context.Context, actorID, orgID primitive.ObjectID, profileRaw, version, reason string) (*domain.CompliancePolicyVersion, error) {
	actorRole, err := s.Tenant.RequireMembership(ctx, actorID, orgID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanPublishPolicy(actorRole) {
		return nil, ErrForbidden
	}

	profile, err := compliance.ValidateProfile(profileRaw)
	if err != nil {
		return nil, err
	}
	if version == "" {
		return nil, fmt.Errorf("%w: version required", compliance.ErrComplianceViolation)
	}

	org, err := s.Orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	rules := compliance.DefaultPlatformPolicyRules(profile)
	policy, err := s.PolicyRepo.PublishOrgPolicy(ctx, orgID, actorID, profile, version, reason, rules)
	if err != nil {
		return nil, err
	}

	if org.ComplianceProfile == string(profile) {
		_ = s.Orgs.UpdateCompliance(ctx, orgID, string(profile), version)
	}

	uid := actorID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &orgID,
		UserID:         &uid,
		EventType:      domain.EventCompliancePolicyPublish,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"profile": string(profile), "version": version},
	})
	_ = s.ConfigAudit.Record(ctx, domain.ConfigAuditLog{
		ChangedBy:  actorID,
		Reason:     reason,
		OldVersion: org.CompliancePolicyVersion,
		NewVersion: version,
		FieldDiff:  map[string]string{"compliancePolicyVersion": version},
		OrgID:      &orgID,
	})
	return policy, nil
}

func parseAssignableRole(raw string) (domain.OrgRole, error) {
	switch domain.OrgRole(strings.ToUpper(strings.TrimSpace(raw))) {
	case domain.RoleAdmin, domain.RoleAnalyst, domain.RoleViewer:
		return domain.OrgRole(strings.ToUpper(strings.TrimSpace(raw))), nil
	default:
		return "", ErrInvalidRole
	}
}
