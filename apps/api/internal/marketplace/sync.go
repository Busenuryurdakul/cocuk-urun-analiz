package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const maxSyncReviews = 50

// SyncResult is the typed outcome of a live product sync operation.
type SyncResult struct {
	Product       *domain.Product
	Updated       bool
	UpdatedFields []string
	Source        string
	SourceURL     string
	SyncedAt      *time.Time
	Warning       string
}

func (s *Service) SyncProductFromSource(ctx context.Context, organizationID, productID, actorID primitive.ObjectID, force bool) (*SyncResult, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanManageMarketplaceImport(role) {
		return nil, ErrForbidden
	}

	prod, err := s.Products.Get(ctx, actorID, organizationID, productID)
	if err != nil {
		return nil, err
	}

	mappings, err := s.Products.Mappings.FindByProductID(ctx, organizationID, productID)
	if err != nil {
		return nil, err
	}

	mapping, err := selectSourceMapping(prod, mappings)
	if err != nil {
		return nil, ErrNoSourceMapping
	}

	return s.syncWithMapping(ctx, organizationID, actorID, prod, mapping, force)
}

func (s *Service) syncWithMapping(ctx context.Context, organizationID, actorID primitive.ObjectID, prod *domain.Product, mapping *domain.ProductSourceMapping, force bool) (*SyncResult, error) {
	sourceURL := strings.TrimSpace(mapping.SourceURL)
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(mapping.SourceProductID)
	}

	if isSyncCooldownActive(prod.LastSyncedAt, s.SyncCooldown, force) {
		return &SyncResult{
			Product:   prod,
			Source:    string(mapping.Source),
			SourceURL: sourceURL,
			Warning:   warningCooldown,
		}, nil
	}

	adapter, ok := s.Registry.Get(mapping.Source)
	if !ok {
		return nil, fmt.Errorf("%w: adapter not found", ErrProviderFetch)
	}

	fetchURL := strings.TrimSpace(mapping.SourceURL)
	if fetchURL == "" {
		return nil, fmt.Errorf("%w: source url missing", ErrProviderFetch)
	}

	result, err := adapter.Fetch(ctx, fetchURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFetch, err)
	}

	if result.Deferred {
		return &SyncResult{
			Product:   prod,
			Source:    string(mapping.Source),
			SourceURL: sourceURL,
			Warning:   deferredWarning(result.DeferredReason),
		}, nil
	}

	if result.Product == nil {
		return nil, fmt.Errorf("%w: empty provider product payload", ErrProviderFetch)
	}

	fetchedAt := result.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}

	return s.applyLiveFetch(ctx, organizationID, actorID, prod, mapping, result, fetchedAt)
}

func (s *Service) applyLiveFetch(ctx context.Context, organizationID, actorID primitive.ObjectID, prod *domain.Product, mapping *domain.ProductSourceMapping, result *FetchResult, fetchedAt time.Time) (*SyncResult, error) {
	source := string(result.Source)
	sourceRecordID := result.SourceProductID
	if sourceRecordID == "" {
		sourceRecordID = mapping.SourceProductID
	}

	merge := mergeProductFromFetch(*prod, *result.Product, source, sourceRecordID, fetchedAt)
	merged := merge.Product
	now := fetchedAt.UTC()
	merged.LastSyncedAt = &now
	merged.LastSyncSource = source

	if err := s.Products.Products.Update(ctx, organizationID, &merged); err != nil {
		return nil, err
	}

	warning := ""
	if reviewWarn := s.persistReviewsFromFetch(ctx, organizationID, merged.ID, result); reviewWarn != "" {
		warning = reviewWarn
	}

	syncedAt := now
	return &SyncResult{
		Product:       &merged,
		Updated:       merge.FieldsChanged,
		UpdatedFields: merge.UpdatedFields,
		Source:        source,
		SourceURL:     strings.TrimSpace(mapping.SourceURL),
		SyncedAt:      &syncedAt,
		Warning:       warning,
	}, nil
}

func (s *Service) persistReviewsFromFetch(ctx context.Context, organizationID, productID primitive.ObjectID, result *FetchResult) string {
	if len(result.Reviews) == 0 {
		return ""
	}
	source := result.Source
	sourceProductID := result.SourceProductID
	limit := len(result.Reviews)
	if limit > maxSyncReviews {
		limit = maxSyncReviews
	}
	var persistErr error
	for _, reviewRow := range result.Reviews[:limit] {
		text, lang, fp := importpkg.NormalizeReviewRow(reviewRow, source, sourceProductID)
		if text == "" {
			continue
		}
		if _, err := s.Reviews.FindByFingerprint(ctx, organizationID, fp); err == nil {
			continue
		} else if !errors.Is(err, repository.ErrNotFound) {
			persistErr = err
			break
		}
		rev := &domain.MarketplaceReview{
			OrganizationID:     organizationID,
			ProductID:          productID,
			SourceType:         domain.SourceTypeMarketplaceReview,
			Source:             source,
			SourceProductID:    sourceProductID,
			SourceReviewID:     reviewRow.SourceReviewID,
			Rating:             reviewRow.Rating,
			ReviewText:         text,
			FetchedAt:          time.Now().UTC(),
			Language:           lang,
			PIIStatus:          domain.PIIClean,
			ModerationStatus:   domain.ModerationPending,
			SpamStatus:         domain.SpamClean,
			DuplicateStatus:    domain.DuplicateUnique,
			QualityStatus:      domain.QualityPending,
			ProvenanceStatus:   domain.ProvenancePartial,
			LicenseStatus:      domain.LicenseUnknown,
			UsageRightsStatus:  domain.UsageRightsUnknown,
			DatasetEligibility: domain.EligibilityQuarantined,
			Fingerprint:        fp,
		}
		if err := s.Reviews.Create(ctx, rev); err != nil && !errors.Is(err, repository.ErrDuplicate) {
			persistErr = err
			break
		}
	}
	if persistErr != nil {
		return warningReviewPersist
	}
	return ""
}

func deferredWarning(reason string) string {
	if reason == domain.DeferredFetchReason || reason == "" {
		return warningDeferredNoAPI
	}
	if reason == blockedProviderContractReason {
		return warningDeferredNoAPI
	}
	return warningDeferredNoAPI
}

func (s *Service) importFromLiveFetch(ctx context.Context, organizationID, actorID primitive.ObjectID, result *FetchResult) (*domain.Product, int, error) {
	if result.Product == nil {
		return nil, 0, fmt.Errorf("%w: empty provider product payload", ErrProviderFetch)
	}
	in := *result.Product
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = result.SourceProductID
	}

	prod, _, err := s.Products.Create(ctx, product.CreateInput{
		OrganizationID:  organizationID,
		ActorID:         actorID,
		Name:            name,
		Brand:           in.Brand,
		Category:        in.Category,
		Description:     in.Description,
		TargetAge:       in.TargetAge,
		Materials:       in.Materials,
		SafetyWarnings:  in.SafetyWarnings,
		CurrentPrice:    stringifyField(in.CurrentPrice),
		OriginalPrice:   stringifyField(in.OriginalPrice),
		Currency:        in.Currency,
		Seller:          in.Seller,
		Rating:          stringifyField(in.Rating),
		ReviewCount:     stringifyField(in.ReviewCount),
		StockStatus:     in.StockStatus,
		SKU:             in.SKU,
		Source:          result.Source,
		SourceProductID: result.SourceProductID,
		SourceURL:       result.SourceURL,
	})
	if err != nil {
		return nil, 0, err
	}

	fetchedAt := result.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	now := fetchedAt.UTC()
	prod.LastSyncedAt = &now
	prod.LastSyncSource = string(result.Source)
	if err := s.Products.Products.Update(ctx, organizationID, prod); err != nil {
		return nil, 0, err
	}

	reviewCount := 0
	if s.persistReviewsFromFetch(ctx, organizationID, prod.ID, result) == "" {
		reviewCount = len(result.Reviews)
		if reviewCount > maxSyncReviews {
			reviewCount = maxSyncReviews
		}
	}
	return prod, 1 + reviewCount, nil
}

func isSyncCooldownActive(lastSyncedAt *time.Time, cooldown time.Duration, force bool) bool {
	if force || lastSyncedAt == nil || cooldown <= 0 {
		return false
	}
	return time.Since(lastSyncedAt.UTC()) < cooldown
}

func stringifyField(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}
