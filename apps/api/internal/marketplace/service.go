package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/fetch"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/storage"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidSource = errors.New("invalid marketplace source")
	ErrImportPayload = errors.New("import payload required")
)

type Service struct {
	Runs           *repository.MarketplaceImportRunRepository
	Reviews        *repository.MarketplaceReviewRepository
	Raw            *repository.RawSourcePayloadRepository
	Products       *product.Service
	DatasetRecords *repository.DatasetRecordRepository
	Registry       *Registry
	Queue          *queue.ImportQueue
	Storage        *storage.S3Client
	Security       *repository.SecurityEventRepository
	Tenant         *tenant.Guard
}

type StartURLImportInput struct {
	OrganizationID primitive.ObjectID
	ActorID        primitive.ObjectID
	SourceURL      string
}

type StartFileImportInput struct {
	OrganizationID primitive.ObjectID
	ActorID        primitive.ObjectID
	Source         domain.MarketplaceSource
	AccessMode     domain.AccessMode
	Filename       string
	Content        []byte
	ContentType    string
}

func (s *Service) StartURLImport(ctx context.Context, input StartURLImportInput) (*domain.MarketplaceImportRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanManageMarketplaceImport(role) {
		return nil, ErrForbidden
	}

	if _, err := fetch.ValidateURL(input.SourceURL, nil); err != nil {
		uid := input.ActorID
		oid := input.OrganizationID
		_ = s.Security.Record(ctx, domain.SecurityEvent{
			OrganizationID: &oid,
			UserID:         &uid,
			EventType:      domain.EventFetchPolicyViolation,
			Severity:       domain.SeverityWarning,
			Details:        map[string]string{"url": input.SourceURL, "error": err.Error()},
		})
		return nil, err
	}

	source, productID, ok := s.Registry.Detect(input.SourceURL)
	if !ok {
		return nil, ErrInvalidSource
	}

	run := &domain.MarketplaceImportRun{
		OrganizationID: input.OrganizationID,
		RequestedBy:    input.ActorID,
		Source:         source,
		SourceURL:      strings.TrimSpace(input.SourceURL),
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := s.Runs.Create(ctx, run); err != nil {
		return nil, err
	}

	if err := s.Queue.Enqueue(ctx, queue.ImportJob{
		ImportRunID:    run.ID.Hex(),
		OrganizationID: input.OrganizationID.Hex(),
	}); err != nil {
		return nil, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventMarketplaceImportQueued,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"importRunId": run.ID.Hex(), "sourceProductId": productID},
	})
	return run, nil
}

func (s *Service) StartFileImport(ctx context.Context, input StartFileImportInput) (*domain.MarketplaceImportRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanManageMarketplaceImport(role) {
		return nil, ErrForbidden
	}
	if len(input.Content) == 0 {
		return nil, ErrImportPayload
	}

	run := &domain.MarketplaceImportRun{
		OrganizationID: input.OrganizationID,
		RequestedBy:    input.ActorID,
		Source:         input.Source,
		AccessMode:     input.AccessMode,
		Status:         domain.ImportPending,
		ImportRef:      input.Filename,
	}
	if err := s.Runs.Create(ctx, run); err != nil {
		return nil, err
	}

	if s.Storage != nil {
		key := storage.TenantImportKey(input.OrganizationID.Hex(), run.ID.Hex(), input.Filename)
		ref, err := s.Storage.PutObject(ctx, key, input.ContentType, input.Content)
		if err == nil {
			run.ImportRef = ref
			_ = s.Runs.Update(ctx, input.OrganizationID, run)
		}
	}

	if err := s.Queue.Enqueue(ctx, queue.ImportJob{
		ImportRunID:    run.ID.Hex(),
		OrganizationID: input.OrganizationID.Hex(),
	}); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) GetRun(ctx context.Context, actorID, organizationID, runID primitive.ObjectID) (*domain.MarketplaceImportRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadMarketplaceImport(role) {
		return nil, ErrForbidden
	}
	return s.Runs.FindByID(ctx, organizationID, runID)
}

func (s *Service) ListReviewsByProduct(ctx context.Context, actorID, organizationID, productID primitive.ObjectID) ([]domain.MarketplaceReview, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadMarketplaceImport(role) {
		return nil, ErrForbidden
	}
	return s.Reviews.ListByProduct(ctx, organizationID, productID)
}

func (s *Service) ListRuns(ctx context.Context, actorID, organizationID primitive.ObjectID, limit int) ([]domain.MarketplaceImportRun, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadMarketplaceImport(role) {
		return nil, ErrForbidden
	}
	return s.Runs.ListByOrg(ctx, organizationID, limit)
}

func (s *Service) ProcessRun(ctx context.Context, organizationID, runID primitive.ObjectID) error {
	run, err := s.Runs.FindByID(ctx, organizationID, runID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	run.Status = domain.ImportRunning
	run.StartedAt = &now
	if err := s.Runs.Update(ctx, organizationID, run); err != nil {
		return err
	}

	switch run.AccessMode {
	case domain.AccessPermittedPublic:
		return s.processURLImport(ctx, organizationID, run)
	case domain.AccessCSVImport, domain.AccessJSONImport, domain.AccessLicensedDataset, domain.AccessUserProvidedFile:
		return s.processDeferredFileImport(ctx, organizationID, run)
	default:
		run.Status = domain.ImportRejectedByPolicy
		run.ErrorCode = "UNSUPPORTED_ACCESS_MODE"
		finished := time.Now().UTC()
		run.FinishedAt = &finished
		return s.Runs.Update(ctx, organizationID, run)
	}
}

func (s *Service) processURLImport(ctx context.Context, organizationID primitive.ObjectID, run *domain.MarketplaceImportRun) error {
	adapter, ok := s.Registry.Get(run.Source)
	if !ok {
		run.Status = domain.ImportFailed
		run.ErrorCode = "ADAPTER_NOT_FOUND"
		return s.finishRun(ctx, organizationID, run)
	}

	result, err := adapter.Fetch(ctx, run.SourceURL)
	if err != nil {
		run.Status = domain.ImportFailed
		run.ErrorMessage = err.Error()
		return s.finishRun(ctx, organizationID, run)
	}

	if result.Deferred {
		run.Status = domain.ImportPartial
		run.ErrorCode = "DEFERRED_WITH_REASON"
		run.ErrorMessage = result.DeferredReason
		run.RecordsSeen = 1
		run.RecordsRejected = 1
		return s.finishRun(ctx, organizationID, run)
	}
	return s.finishRun(ctx, organizationID, run)
}

func (s *Service) processDeferredFileImport(ctx context.Context, organizationID primitive.ObjectID, run *domain.MarketplaceImportRun) error {
	now := time.Now().UTC()
	run.Status = domain.ImportRunning
	run.StartedAt = &now
	if err := s.Runs.Update(ctx, organizationID, run); err != nil {
		return err
	}

	var content []byte
	var err error
	if s.Storage != nil && strings.HasPrefix(run.ImportRef, "s3://") {
		key, keyErr := parseObjectFromImportRef(run.ImportRef)
		if keyErr != nil {
			run.Status = domain.ImportFailed
			run.ErrorCode = "INVALID_IMPORT_REF"
			run.ErrorMessage = keyErr.Error()
			return s.finishRun(ctx, organizationID, run)
		}
		content, err = s.Storage.GetObject(ctx, key)
	} else {
		run.Status = domain.ImportFailed
		run.ErrorCode = "IMPORT_PAYLOAD_NOT_FOUND"
		run.ErrorMessage = "stored import payload unavailable"
		return s.finishRun(ctx, organizationID, run)
	}
	if err != nil {
		run.Status = domain.ImportFailed
		run.ErrorCode = "STORAGE_FETCH_FAILED"
		run.ErrorMessage = err.Error()
		return s.finishRun(ctx, organizationID, run)
	}

	filename := run.ImportRef
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		filename = filename[idx+1:]
	}
	parsed, err := parseImportPayload(content, filename)
	if err != nil {
		run.Status = domain.ImportFailed
		run.ErrorCode = "PARSE_FAILED"
		run.ErrorMessage = err.Error()
		return s.finishRun(ctx, organizationID, run)
	}

	stats, err := s.importParseResult(ctx, organizationID, run.RequestedBy, run.Source, parsed)
	run.RecordsSeen = stats.ProductsAccepted + stats.ReviewsAccepted + stats.ReviewsSkipped
	run.RecordsAccepted = stats.ProductsAccepted + stats.ReviewsAccepted
	run.RecordsRejected = stats.ReviewsSkipped
	run.RecordsQuarantined = stats.RecordsQuarantined
	if err != nil {
		run.Status = domain.ImportFailed
		run.ErrorCode = "PERSIST_FAILED"
		run.ErrorMessage = err.Error()
		return s.finishRun(ctx, organizationID, run)
	}
	if stats.ProductsAccepted == 0 {
		run.Status = domain.ImportFailed
		run.ErrorCode = "NO_PRODUCTS_IMPORTED"
		return s.finishRun(ctx, organizationID, run)
	}
	run.Status = domain.ImportSucceeded
	return s.finishRun(ctx, organizationID, run)
}

func (s *Service) finishRun(ctx context.Context, organizationID primitive.ObjectID, run *domain.MarketplaceImportRun) error {
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	return s.Runs.Update(ctx, organizationID, run)
}

// ImportParsedProducts persists parsed CSV/JSON products — used by consumer when raw payload available.
func (s *Service) ImportParsedProducts(ctx context.Context, organizationID, actorID primitive.ObjectID, source domain.MarketplaceSource, result *importpkg.ParseResult) (int, error) {
	stats, err := s.importParseResult(ctx, organizationID, actorID, source, result)
	if err != nil {
		return 0, err
	}
	return stats.ProductsAccepted, nil
}
