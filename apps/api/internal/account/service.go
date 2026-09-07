package account

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
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrAccountDeleted            = errors.New("account deleted")
	ErrOwnershipTransferRequired = errors.New("ownership transfer required")
	ErrInvalidDeletionCode       = errors.New("invalid deletion code")
)

type DeletionRequestResult struct {
	Status    string
	ExpiresAt time.Time
}

type Service struct {
	Users        *repository.UserRepository
	Orgs         *repository.OrganizationRepository
	Members      *repository.MemberRepository
	Devices      *repository.DeviceRepository
	Sessions     *repository.SessionRepository
	DeletionReqs *repository.AccountDeletionRepository
	Consents     *repository.ConsentRepository
	AnalysisRuns *repository.AnalysisRunRepository
	ActivityLogs *repository.UserActivityLogRepository
	Security     *repository.SecurityEventRepository
	Mail         mail.Service
	Policy       auth.SecurityPolicy
}

func (s *Service) RequestDeletion(ctx context.Context, userID primitive.ObjectID) (*DeletionRequestResult, error) {
	user, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, ErrAccountDeleted
	}

	code, codeHash, err := auth.NewNumericCode()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(s.Policy.AccountDeletionTTL)
	req := &domain.AccountDeletionRequest{
		UserID:    userID,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
	}
	if err := s.DeletionReqs.ReplacePending(ctx, req); err != nil {
		return nil, err
	}

	subject, plain, html := mail.AccountDeletionEmail(code)
	if err := s.Mail.Send(ctx, mail.Message{
		To:       user.Email,
		Subject:  subject,
		Body:     plain,
		HTMLBody: html,
	}); err != nil {
		return nil, err
	}

	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventAccountDeletionRequested,
		Severity:  domain.SeverityWarning,
		Details:   map[string]string{"stage": "pending_confirmation"},
	})

	return &DeletionRequestResult{
		Status:    "PENDING_CONFIRMATION",
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) ConfirmDeletion(ctx context.Context, userID primitive.ObjectID, code string) error {
	user, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.IsDeleted() {
		return nil
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return ErrInvalidDeletionCode
	}
	codeHash := auth.HashToken(code)
	if _, err := s.DeletionReqs.ConsumeByCodeHash(ctx, userID, codeHash); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidDeletionCode
		}
		return err
	}

	if blocked, err := s.hasBlockingOwnership(ctx, userID); err != nil {
		return err
	} else if blocked {
		uid := userID
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			UserID:    &uid,
			EventType: domain.EventAccountDeletionBlockedOwnership,
			Severity:  domain.SeverityWarning,
			Details:   map[string]string{"reason": "ownership_transfer_required"},
		})
		return ErrOwnershipTransferRequired
	}

	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventAccountDeletionConfirmed,
		Severity:  domain.SeverityWarning,
		Details:   map[string]string{"stage": "executing"},
	})

	if err := s.Sessions.RevokeAllByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.Devices.RevokeAllByUser(ctx, userID); err != nil {
		return err
	}
	if err := s.cleanupMemberships(ctx, userID); err != nil {
		return err
	}
	if err := s.withdrawConsents(ctx, userID); err != nil {
		return err
	}
	if err := s.AnalysisRuns.AnonymizeUserActor(ctx, userID); err != nil {
		return err
	}

	anonEmail := fmt.Sprintf("deleted+%s@anonymized.invalid", userID.Hex())
	if err := s.Users.AnonymizeDeleted(ctx, userID, anonEmail); err != nil {
		return err
	}
	_ = s.DeletionReqs.DeleteByUserID(ctx, userID)

	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventAccountDeleted,
		Severity:  domain.SeverityInfo,
		Details: map[string]string{
			"retention": "audit_records_retained_pseudonymized",
		},
	})
	return nil
}

func (s *Service) hasBlockingOwnership(ctx context.Context, userID primitive.ObjectID) (bool, error) {
	memberships, err := s.Members.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, m := range memberships {
		if m.Role != domain.RoleOwner {
			continue
		}
		others, err := s.Members.CountMembersExcluding(ctx, m.OrganizationID, userID)
		if err != nil {
			return false, err
		}
		if others > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) cleanupMemberships(ctx context.Context, userID primitive.ObjectID) error {
	memberships, err := s.Members.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, m := range memberships {
		others, err := s.Members.CountMembersExcluding(ctx, m.OrganizationID, userID)
		if err != nil {
			return err
		}
		if err := s.Members.Delete(ctx, m.OrganizationID, userID); err != nil {
			return err
		}
		if others == 0 {
			if err := s.Orgs.Delete(ctx, m.OrganizationID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) withdrawConsents(ctx context.Context, userID primitive.ObjectID) error {
	consents, err := s.Consents.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, c := range consents {
		if c.WithdrawnAt != nil {
			continue
		}
		key := fmt.Sprintf("%s:%s:%s", c.Purpose, orgIDKey(c.OrganizationID), c.PolicyVersion)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := s.Consents.Withdraw(ctx, userID, c.OrganizationID, c.Purpose); err != nil {
			return err
		}
	}
	return nil
}

func orgIDKey(orgID *primitive.ObjectID) string {
	if orgID == nil {
		return "platform"
	}
	return orgID.Hex()
}

func (s *Service) ExportData(ctx context.Context, userID primitive.ObjectID) (*ExportDocument, error) {
	user, err := s.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, ErrAccountDeleted
	}

	uid := userID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		UserID:    &uid,
		EventType: domain.EventDataExportRequested,
		Severity:  domain.SeverityInfo,
		Details:   map[string]string{"schemaVersion": ExportSchemaVersion},
	})

	doc, err := s.buildExport(ctx, user)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) buildExport(ctx context.Context, user *domain.User) (*ExportDocument, error) {
	consents, err := s.Consents.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	memberships, err := s.Members.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	devices, err := s.Devices.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	runs, err := s.AnalysisRuns.ListByCreatedByUser(ctx, user.ID, 200)
	if err != nil {
		return nil, err
	}
	activity, err := s.ActivityLogs.ListByUser(ctx, user.ID, 100, nil)
	if err != nil {
		return nil, err
	}

	exportConsents := make([]ExportConsent, 0, len(consents))
	for _, c := range consents {
		item := ExportConsent{
			Purpose:       string(c.Purpose),
			PolicyVersion: c.PolicyVersion,
			GrantedAt:     c.GrantedAt.UTC().Format(time.RFC3339),
		}
		if c.OrganizationID != nil {
			id := c.OrganizationID.Hex()
			item.OrganizationID = &id
		}
		if c.WithdrawnAt != nil {
			w := c.WithdrawnAt.UTC().Format(time.RFC3339)
			item.WithdrawnAt = &w
		}
		exportConsents = append(exportConsents, item)
	}

	exportMemberships := make([]ExportMembership, 0, len(memberships))
	for _, m := range memberships {
		org, orgErr := s.Orgs.FindByID(ctx, m.OrganizationID)
		name := ""
		orgType := string(domain.OrgTypeOrganization)
		if orgErr == nil && org != nil {
			name = compliance.SanitizeForAudit(org.Name)
			orgType = string(org.Type)
		}
		exportMemberships = append(exportMemberships, ExportMembership{
			OrganizationID:   m.OrganizationID.Hex(),
			OrganizationName: name,
			OrganizationType: orgType,
			Role:             string(m.Role),
			JoinedAt:         m.JoinedAt.UTC().Format(time.RFC3339),
		})
	}

	exportDevices := make([]ExportDevice, 0, len(devices))
	for _, d := range devices {
		exportDevices = append(exportDevices, ExportDevice{
			ID:           d.ID.Hex(),
			Platform:     string(d.Platform),
			Label:        d.Label,
			Verified:     d.Verified,
			LastActiveAt: d.LastActiveAt.UTC().Format(time.RFC3339),
			CreatedAt:    d.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	exportRuns := make([]ExportAnalysisActivity, 0, len(runs))
	for _, run := range runs {
		exportRuns = append(exportRuns, ExportAnalysisActivity{
			RunID:           run.ID.Hex(),
			OrganizationID:  run.OrganizationID.Hex(),
			Status:          string(run.Status),
			ClientRequestID: run.ClientRequestID,
			CreatedAt:       run.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	exportActivity := make([]ExportActivityEntry, 0, len(activity))
	for _, entry := range activity {
		item := ExportActivityEntry{
			Action:    entry.Action,
			Timestamp: entry.Timestamp.UTC().Format(time.RFC3339),
		}
		if entry.OrganizationID != nil {
			id := entry.OrganizationID.Hex()
			item.OrganizationID = &id
		}
		exportActivity = append(exportActivity, item)
	}

	return &ExportDocument{
		SchemaVersion: ExportSchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		User: ExportUserProfile{
			ID:            user.ID.Hex(),
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			MFAEnabled:    user.MFAEnabled,
			PersonalOrgID: user.PersonalOrgID.Hex(),
			CreatedAt:     user.CreatedAt.UTC().Format(time.RFC3339),
		},
		Consents:         exportConsents,
		Memberships:      exportMemberships,
		Devices:          exportDevices,
		AnalysisActivity: exportRuns,
		ActivityLog:      exportActivity,
	}, nil
}
