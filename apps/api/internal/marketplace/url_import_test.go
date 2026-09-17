package marketplace

import (
	"errors"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProcessURLImportNewURLCreatesProductAndMapping(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("299")
	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}

	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatalf("processURLImport: %v", err)
	}
	if run.Status != domain.ImportSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s (%s)", run.Status, run.ErrorMessage)
	}
	if run.RecordsAccepted == 0 {
		t.Fatal("expected accepted records")
	}

	mapping, err := h.products.Mappings.FindBySourceProduct(h.ctx, h.orgID, domain.MarketplaceTrendyol, "424242")
	if err != nil {
		t.Fatalf("mapping: %v", err)
	}
	prod, err := h.productRepo.FindByID(h.ctx, h.orgID, mapping.ProductID)
	if err != nil {
		t.Fatalf("product: %v", err)
	}
	if prod.LastSyncedAt == nil {
		t.Fatal("expected lastSyncedAt on new import")
	}
	if prod.Name.Value == "" {
		t.Fatal("placeholder product must not be created")
	}
}

func TestProcessURLImportExistingMappingUpdatesProduct(t *testing.T) {
	h := newSyncHarness(t)
	prod := h.createProductWithMapping("100", nil)
	h.successFetch("175")

	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}

	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatalf("processURLImport: %v", err)
	}
	if run.Status != domain.ImportSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", run.Status)
	}

	got, err := h.productRepo.FindByID(h.ctx, h.orgID, prod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentPrice.Value != "175" {
		t.Fatalf("expected merged price 175, got %v", got.CurrentPrice.Value)
	}
	mappings, err := h.products.Mappings.FindByProductID(h.ctx, h.orgID, prod.ID)
	if err != nil || len(mappings) != 1 {
		t.Fatalf("duplicate mapping created: %v", err)
	}
}

func TestProcessURLImportDeferredPartialNoProduct(t *testing.T) {
	h := newSyncHarness(t)
	h.deferredFetch()
	beforeCount, _ := h.productRepo.ListByOrg(h.ctx, h.orgID, 100)

	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatalf("processURLImport: %v", err)
	}
	if run.Status != domain.ImportPartial {
		t.Fatalf("expected PARTIAL, got %s", run.Status)
	}
	if run.Status == domain.ImportSucceeded {
		t.Fatal("deferred must not succeed")
	}
	if run.ErrorMessage != warningDeferredNoAPI {
		t.Fatalf("unexpected deferred message %q", run.ErrorMessage)
	}
	after, _ := h.productRepo.ListByOrg(h.ctx, h.orgID, 100)
	if len(after) != len(beforeCount) {
		t.Fatal("deferred import must not create canonical product")
	}
}

func TestProcessURLImportUsesOrganizationAndActor(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("120")
	otherActor := primitive.NewObjectID()
	if err := h.svc.Tenant.Members.Create(h.ctx, &domain.OrganizationMember{
		OrganizationID: h.orgID,
		UserID:         otherActor,
		Role:           domain.RoleAnalyst,
	}); err != nil {
		t.Fatal(err)
	}

	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    otherActor,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatal(err)
	}
	mapping, err := h.products.Mappings.FindBySourceProduct(h.ctx, h.orgID, domain.MarketplaceTrendyol, "424242")
	if err != nil {
		t.Fatal(err)
	}
	prod, err := h.productRepo.FindByID(h.ctx, h.orgID, mapping.ProductID)
	if err != nil {
		t.Fatal(err)
	}
	if prod.CreatedBy != otherActor {
		t.Fatalf("expected actor %s, got %s", otherActor.Hex(), prod.CreatedBy.Hex())
	}
	if prod.OrganizationID != h.orgID {
		t.Fatal("tenant organization mismatch")
	}
}

func TestProcessURLImportWrongTenantOrganizationRejected(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("120")
	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.otherOrgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.otherOrgID, run); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != domain.ImportFailed {
		t.Fatalf("expected FAILED when actor not in tenant org, got %s", run.Status)
	}
}

func TestProcessURLImportProviderErrorFailsRun(t *testing.T) {
	h := newSyncHarness(t)
	h.errorFetch(errors.New("upstream timeout"))
	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatalf("processURLImport: %v", err)
	}
	if run.Status != domain.ImportFailed {
		t.Fatalf("expected FAILED, got %s", run.Status)
	}
}

func TestProcessURLImportReviewDedup(t *testing.T) {
	h := newSyncHarness(t)
	h.fetchWithReviews()
	run := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.orgID, run); err != nil {
		t.Fatal(err)
	}
	run2 := &domain.MarketplaceImportRun{
		ID:             primitive.NewObjectID(),
		OrganizationID: h.orgID,
		RequestedBy:    h.actorID,
		Source:         domain.MarketplaceTrendyol,
		SourceURL:      testTrendyolURL,
		AccessMode:     domain.AccessPermittedPublic,
		Status:         domain.ImportPending,
	}
	if err := h.svc.processURLImport(h.ctx, h.orgID, run2); err != nil {
		t.Fatal(err)
	}
	mapping, _ := h.products.Mappings.FindBySourceProduct(h.ctx, h.orgID, domain.MarketplaceTrendyol, "424242")
	reviews, err := h.svc.Reviews.ListByProduct(h.ctx, h.orgID, mapping.ProductID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected deduped single review, got %d", len(reviews))
	}
}
