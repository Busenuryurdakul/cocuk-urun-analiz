package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSyncProductFromSourceSuccessfulUpdate(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("199")
	prod := h.createProductWithMapping("100", nil)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !result.Updated {
		t.Fatal("expected updated=true")
	}
	if len(result.UpdatedFields) == 0 || !containsField(result.UpdatedFields, "price") {
		t.Fatalf("expected price in updatedFields, got %v", result.UpdatedFields)
	}
	if result.SyncedAt == nil {
		t.Fatal("expected syncedAt")
	}
	if result.Source != string(domain.MarketplaceTrendyol) {
		t.Fatalf("unexpected source %q", result.Source)
	}
	got := h.reloadProduct(prod.ID)
	if got.LastSyncedAt == nil {
		t.Fatal("expected lastSyncedAt on product")
	}
	if got.CurrentPrice.Value != "199" {
		t.Fatalf("expected price 199, got %v", got.CurrentPrice.Value)
	}
	if got.CurrentPrice.Source == "" {
		t.Fatal("expected provenance on changed field")
	}
}

func TestSyncProductFromSourceSuccessfulNoOpStillUpdatesLastSyncedAt(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetchNoOp("100")
	prod := h.createProductWithMapping("100", nil)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Updated {
		t.Fatal("expected updated=false for no-op merge")
	}
	if len(result.UpdatedFields) != 0 {
		t.Fatalf("expected empty updatedFields, got %v", result.UpdatedFields)
	}
	if result.SyncedAt == nil {
		t.Fatal("expected syncedAt on successful fetch")
	}
	got := h.reloadProduct(prod.ID)
	if got.LastSyncedAt == nil {
		t.Fatal("expected lastSyncedAt updated even on no-op")
	}
}

func TestSyncProductFromSourceDeferredNoWrite(t *testing.T) {
	h := newSyncHarness(t)
	h.deferredFetch()
	prod := h.createProductWithMapping("100", nil)
	before := h.reloadProduct(prod.ID)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.SyncedAt != nil {
		t.Fatal("expected syncedAt null on deferred")
	}
	if result.Warning != warningDeferredNoAPI {
		t.Fatalf("unexpected warning %q", result.Warning)
	}
	after := h.reloadProduct(prod.ID)
	if after.LastSyncedAt != before.LastSyncedAt {
		t.Fatal("deferred must not change lastSyncedAt")
	}
	if after.CurrentPrice.Value != before.CurrentPrice.Value {
		t.Fatal("deferred must not change canonical fields")
	}
}

func TestSyncProductFromSourceCooldownSkipsProvider(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("250")
	recent := time.Now().UTC()
	prod := h.createProductWithMapping("100", &recent)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if h.mock.Calls() != 0 {
		t.Fatalf("expected 0 provider calls during cooldown, got %d", h.mock.Calls())
	}
	if result.SyncedAt != nil {
		t.Fatal("expected syncedAt null on cooldown")
	}
	if !strings.Contains(result.Warning, "kısa süre önce") {
		t.Fatalf("unexpected cooldown warning %q", result.Warning)
	}
}

func TestSyncProductFromSourceForceBypassesCooldown(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("250")
	recent := time.Now().UTC()
	prod := h.createProductWithMapping("100", &recent)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, true)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if h.mock.Calls() != 1 {
		t.Fatalf("expected provider call with force=true, got %d", h.mock.Calls())
	}
	if result.SyncedAt == nil {
		t.Fatal("expected syncedAt after forced fetch")
	}
}

func TestSyncProductFromSourceProviderError(t *testing.T) {
	h := newSyncHarness(t)
	h.errorFetch(fmt.Errorf("timeout"))
	prod := h.createProductWithMapping("100", nil)

	_, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err == nil || !errors.Is(err, ErrProviderFetch) {
		t.Fatalf("expected provider fetch error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "api_key") {
		t.Fatal("secrets must not leak into errors")
	}
}

func TestSyncProductFromSourceProvider5xx(t *testing.T) {
	h := newSyncHarness(t)
	h.errorFetch(fmt.Errorf("upstream status 503"))
	prod := h.createProductWithMapping("100", nil)

	_, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err == nil || !errors.Is(err, ErrProviderFetch) {
		t.Fatalf("expected provider fetch error, got %v", err)
	}
}

func TestSyncProductFromSourceMappingNotFound(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("100")
	prod, _, err := h.products.Create(h.ctx, product.CreateInput{
		OrganizationID:  h.orgID,
		ActorID:         h.actorID,
		Name:            "No Mapping Product",
		Source:          domain.MarketplaceMiyuna,
		SourceProductID: "no-mapping-" + primitive.NewObjectID().Hex(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err == nil || !errors.Is(err, ErrNoSourceMapping) {
		t.Fatalf("expected mapping not found error, got %v", err)
	}
}

func TestSyncProductFromSourceViewerForbidden(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("100")
	prod := h.createProductWithMapping("100", nil)

	_, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.viewerID, false)
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for viewer, got %v", err)
	}
}

func TestSyncProductFromSourceCrossTenantProduct(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("100")
	prod := h.createProductWithMapping("100", nil)

	_, err := h.svc.SyncProductFromSource(h.ctx, h.otherOrgID, prod.ID, h.actorID, false)
	if err == nil {
		t.Fatal("expected cross-tenant rejection")
	}
}

func TestSyncProductFromSourceReviewSuccessDoesNotFailSync(t *testing.T) {
	h := newSyncHarness(t)
	h.fetchWithReviews()
	prod := h.createProductWithMapping("100", nil)

	result, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, prod.ID, h.actorID, false)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.SyncedAt == nil {
		t.Fatal("expected successful sync with reviews")
	}
	if result.Warning != "" {
		t.Fatalf("unexpected warning %q", result.Warning)
	}
}

func TestPersistReviewsFromFetchFailureReturnsWarning(t *testing.T) {
	h := newSyncHarness(t)
	ctx, cancel := context.WithCancel(h.ctx)
	cancel()
	rating := 4.0
	result := &FetchResult{
		Source:          domain.MarketplaceTrendyol,
		SourceProductID: "424242",
		Reviews: []importpkg.ReviewRow{{
			ReviewText: "review text",
			Rating:     &rating,
		}},
	}
	warn := h.svc.persistReviewsFromFetch(ctx, h.orgID, primitive.NewObjectID(), result)
	if warn != warningReviewPersist {
		t.Fatalf("expected review persist warning, got %q", warn)
	}
}

func containsField(fields []string, slug string) bool {
	for _, f := range fields {
		if f == slug {
			return true
		}
	}
	return false
}

func TestSyncProductFromSourceProductNotFound(t *testing.T) {
	h := newSyncHarness(t)
	h.successFetch("100")
	missingID := primitive.NewObjectID()

	_, err := h.svc.SyncProductFromSource(h.ctx, h.orgID, missingID, h.actorID, false)
	if err == nil || !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
