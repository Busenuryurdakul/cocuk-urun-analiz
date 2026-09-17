package marketplace

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const testTrendyolURL = "https://www.trendyol.com/sync-test-p-424242"

type mockFetchAdapter struct {
	source     domain.MarketplaceSource
	fetchFn    func(ctx context.Context, rawURL string) (*FetchResult, error)
	fetchCalls atomic.Int32
}

func (m *mockFetchAdapter) Source() domain.MarketplaceSource {
	if m.source == "" {
		return domain.MarketplaceTrendyol
	}
	return m.source
}

func (m *mockFetchAdapter) DetectURL(rawURL string) (bool, string) {
	return true, "424242"
}

func (m *mockFetchAdapter) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	m.fetchCalls.Add(1)
	if m.fetchFn != nil {
		return m.fetchFn(ctx, rawURL)
	}
	return nil, fmt.Errorf("mock fetch not configured")
}

func (m *mockFetchAdapter) Calls() int {
	return int(m.fetchCalls.Load())
}

type syncHarness struct {
	t           *testing.T
	ctx         context.Context
	cancel      context.CancelFunc
	svc         *Service
	products    *product.Service
	productRepo *repository.ProductRepository
	mock        *mockFetchAdapter
	orgID       primitive.ObjectID
	actorID     primitive.ObjectID
	viewerID    primitive.ObjectID
	otherOrgID  primitive.ObjectID
}

func newSyncHarness(t *testing.T) *syncHarness {
	t.Helper()
	if !mongoTestsEnabled {
		t.Skip("mongodb unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	dbName := fmt.Sprintf("miyuna_sync_test_%d", time.Now().UnixNano())
	client, err := mongoclient.Connect(ctx, fmt.Sprintf("mongodb://localhost:27017/%s", dbName))
	if err != nil {
		cancel()
		t.Skipf("mongodb unavailable: %v", err)
	}

	db := client.DB
	members := repository.NewMemberRepository(db)
	events := repository.NewSecurityEventRepository(db)
	guard := &tenant.Guard{Members: members, Events: events}
	productRepo := repository.NewProductRepository(db)
	mappingRepo := repository.NewProductSourceMappingRepository(db)
	reviewRepo := repository.NewMarketplaceReviewRepository(db)
	runsRepo := repository.NewMarketplaceImportRunRepository(db)

	products := &product.Service{
		Products: productRepo,
		Mappings: mappingRepo,
		Security: events,
		Tenant:   guard,
	}

	mock := &mockFetchAdapter{source: domain.MarketplaceTrendyol}
	svc := &Service{
		Runs:             runsRepo,
		Reviews:          reviewRepo,
		Products:         products,
		Registry:         NewRegistry(mock),
		Tenant:           guard,
		SyncCooldown:     time.Hour,
		ProviderSettings: ProviderSettings{},
	}

	h := &syncHarness{
		t:           t,
		ctx:         ctx,
		cancel:      cancel,
		svc:         svc,
		products:    products,
		productRepo: productRepo,
		mock:        mock,
		orgID:       primitive.NewObjectID(),
		actorID:     primitive.NewObjectID(),
		viewerID:    primitive.NewObjectID(),
		otherOrgID:  primitive.NewObjectID(),
	}
	t.Cleanup(func() {
		cancel()
		_ = client.Disconnect(context.Background())
	})

	for _, spec := range []struct {
		user primitive.ObjectID
		org  primitive.ObjectID
		role domain.OrgRole
	}{
		{h.actorID, h.orgID, domain.RoleAnalyst},
		{h.viewerID, h.orgID, domain.RoleViewer},
	} {
		if err := members.Create(h.ctx, &domain.OrganizationMember{
			OrganizationID: spec.org,
			UserID:         spec.user,
			Role:           spec.role,
		}); err != nil {
			t.Fatal(err)
		}
	}

	return h
}

func (h *syncHarness) liveProductInput(price string) *normalize.ProductInput {
	priceVal := any(price)
	if price == "" {
		priceVal = nil
	}
	return &normalize.ProductInput{
		Name:         "Live Product",
		Brand:        "TestBrand",
		CurrentPrice: priceVal,
		Rating:       4.2,
		ReviewCount:  10,
		StockStatus:  "in_stock",
	}
}

func (h *syncHarness) successFetch(price string) {
	h.mock.fetchFn = func(ctx context.Context, rawURL string) (*FetchResult, error) {
		return &FetchResult{
			Source:          domain.MarketplaceTrendyol,
			SourceURL:       rawURL,
			SourceProductID: "424242",
			Product:         h.liveProductInput(price),
			FetchedAt:       time.Now().UTC(),
		}, nil
	}
}

func (h *syncHarness) deferredFetch() {
	h.mock.fetchFn = func(ctx context.Context, rawURL string) (*FetchResult, error) {
		return &FetchResult{
			Deferred:        true,
			DeferredReason:  domain.DeferredFetchReason,
			Source:          domain.MarketplaceTrendyol,
			SourceURL:       rawURL,
			SourceProductID: "424242",
		}, nil
	}
}

func (h *syncHarness) errorFetch(err error) {
	h.mock.fetchFn = func(ctx context.Context, rawURL string) (*FetchResult, error) {
		return nil, err
	}
}

func (h *syncHarness) createProductWithMapping(price string, lastSynced *time.Time) *domain.Product {
	h.t.Helper()
	prod, _, err := h.products.Create(h.ctx, product.CreateInput{
		OrganizationID:  h.orgID,
		ActorID:         h.actorID,
		Name:            "Canonical Product",
		Brand:           "LocalBrand",
		CurrentPrice:    price,
		Rating:          "4.0",
		ReviewCount:     "5",
		StockStatus:     "in_stock",
		Source:          domain.MarketplaceTrendyol,
		SourceProductID: "424242",
		SourceURL:       testTrendyolURL,
	})
	if err != nil {
		h.t.Fatal(err)
	}
	if lastSynced != nil {
		prod.LastSyncedAt = lastSynced
		prod.LastSyncSource = string(domain.MarketplaceTrendyol)
		if err := h.productRepo.Update(h.ctx, h.orgID, prod); err != nil {
			h.t.Fatal(err)
		}
	}
	return prod
}

func (h *syncHarness) reloadProduct(productID primitive.ObjectID) *domain.Product {
	h.t.Helper()
	prod, err := h.productRepo.FindByID(h.ctx, h.orgID, productID)
	if err != nil {
		h.t.Fatal(err)
	}
	return prod
}

func (h *syncHarness) fetchWithReviews() {
	h.mock.fetchFn = func(ctx context.Context, rawURL string) (*FetchResult, error) {
		rating := 5.0
		return &FetchResult{
			Source:          domain.MarketplaceTrendyol,
			SourceURL:       rawURL,
			SourceProductID: "424242",
			Product:         h.liveProductInput("150"),
			Reviews: []importpkg.ReviewRow{{
				ReviewText: "Great toy",
				Rating:     &rating,
			}},
			FetchedAt: time.Now().UTC(),
		}, nil
	}
}
