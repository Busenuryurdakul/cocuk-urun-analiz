package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const gqlSyncMutation = `mutation($input: SyncProductFromSourceInput!) {
	syncProductFromSource(input: $input) {
		updated
		updatedFields
		source
		syncedAt
		warning
		product { id lastSyncedAt }
	}
}`

func TestGraphQLSyncProductUnauthenticated(t *testing.T) {
	h := NewPhase5Harness(t)
	_, errs, _ := h.GraphQL("", gqlSyncMutation, map[string]any{
		"input": map[string]any{
			"organizationId": h.OrgID.Hex(),
			"productId":      h.ProductID.Hex(),
			"force":          false,
		},
	})
	if !errHasCode(errs, "UNAUTHORIZED") && len(errs) == 0 {
		t.Fatalf("expected unauthenticated rejection, got %v", errs)
	}
}

func TestGraphQLSyncProductViewerForbidden(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	token := h.AccessToken(h.ViewerID, h.OrgID)
	_, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for viewer, got %v", errs)
	}
}

func TestGraphQLSyncProductAnalystAllowedDeferred(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	before, _ := h.App.Products.Get(h.Ctx, h.AnalystID, h.OrgID, h.ProductID)

	data, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if len(errs) > 0 {
		t.Fatalf("deferred should be success payload: %v", errs)
	}
	payload := data["syncProductFromSource"].(map[string]any)
	if payload["syncedAt"] != nil {
		t.Fatalf("expected null syncedAt on deferred, got %v", payload["syncedAt"])
	}
	if payload["updated"] != false {
		t.Fatal("expected updated=false")
	}
	warning, _ := payload["warning"].(string)
	if warning == "" {
		t.Fatal("expected deferred warning")
	}

	after, _ := h.App.Products.Get(h.Ctx, h.AnalystID, h.OrgID, h.ProductID)
	if fmt.Sprint(before.LastSyncedAt) != fmt.Sprint(after.LastSyncedAt) {
		t.Fatal("deferred must not change lastSyncedAt")
	}
}

func TestGraphQLSyncProductCrossTenantOrgRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OtherOrgID, h.ProductID, false))
	if !errHasCode(errs, "FORBIDDEN") {
		t.Fatalf("expected FORBIDDEN for org mismatch, got %v", errs)
	}
}

func TestGraphQLSyncProductCrossTenantProductRejected(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.OtherProdID, false))
	if len(errs) == 0 {
		t.Fatal("expected error for other tenant product")
	}
}

func TestGraphQLSyncProductMappingNotFound(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if !errHasCode(errs, "NO_SOURCE_MAPPING") {
		t.Fatalf("expected NO_SOURCE_MAPPING, got %v", errs)
	}
}

func TestGraphQLSyncProductCooldownPayload(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	recent := time.Now().UTC()
	prod, err := h.App.Products.Products.FindByID(h.Ctx, h.OrgID, h.ProductID)
	if err != nil {
		t.Fatal(err)
	}
	prod.LastSyncedAt = &recent
	prod.LastSyncSource = string(domain.MarketplaceTrendyol)
	if err := h.App.Products.Products.Update(h.Ctx, h.OrgID, prod); err != nil {
		t.Fatal(err)
	}
	h.App.Marketplace.SyncCooldown = time.Hour

	token := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if len(errs) > 0 {
		t.Fatalf("cooldown should be success payload: %v", errs)
	}
	payload := data["syncProductFromSource"].(map[string]any)
	if payload["syncedAt"] != nil {
		t.Fatal("expected null syncedAt on cooldown")
	}
	warning, _ := payload["warning"].(string)
	if warning == "" {
		t.Fatal("expected cooldown warning")
	}
}

func TestGraphQLSyncProductSuccessfulUpdateRFC3339(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	h.App.Marketplace.Registry = marketplace.NewRegistry(&gqlMockAdapter{})
	h.App.Marketplace.SyncCooldown = 0

	token := h.AccessToken(h.AnalystID, h.OrgID)
	data, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if len(errs) > 0 {
		t.Fatalf("sync: %v", errs)
	}
	payload := data["syncProductFromSource"].(map[string]any)
	syncedAt, ok := payload["syncedAt"].(string)
	if !ok || syncedAt == "" {
		t.Fatalf("expected RFC3339 syncedAt string, got %v", payload["syncedAt"])
	}
	if _, err := time.Parse(time.RFC3339, syncedAt); err != nil {
		t.Fatalf("syncedAt not RFC3339: %v", err)
	}
	productPayload := payload["product"].(map[string]any)
	lastSynced, ok := productPayload["lastSyncedAt"].(string)
	if !ok || lastSynced == "" {
		t.Fatal("expected product.lastSyncedAt")
	}
	if _, err := time.Parse(time.RFC3339, lastSynced); err != nil {
		t.Fatalf("lastSyncedAt not RFC3339: %v", err)
	}
}

func TestGraphQLSyncProductProviderError(t *testing.T) {
	h := NewPhase5Harness(t)
	seedTrendyolMapping(t, h, h.ProductID)
	h.App.Marketplace.Registry = marketplace.NewRegistry(&gqlMockAdapter{fail: true})
	h.App.Marketplace.SyncCooldown = 0

	token := h.AccessToken(h.AnalystID, h.OrgID)
	_, errs, _ := h.GraphQL(token, gqlSyncMutation, syncInput(h.OrgID, h.ProductID, false))
	if !errHasCode(errs, "PROVIDER_FETCH_FAILED") {
		t.Fatalf("expected PROVIDER_FETCH_FAILED, got %v", errs)
	}
}

func syncInput(orgID, productID primitive.ObjectID, force bool) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"organizationId": orgID.Hex(),
			"productId":      productID.Hex(),
			"force":          force,
		},
	}
}

func seedTrendyolMapping(t *testing.T, h *Phase5Harness, productID primitive.ObjectID) {
	t.Helper()
	err := h.App.Products.Mappings.Create(h.Ctx, &domain.ProductSourceMapping{
		OrganizationID:  h.OrgID,
		ProductID:       productID,
		Source:          domain.MarketplaceTrendyol,
		SourceProductID: "987654",
		SourceURL:       "https://www.trendyol.com/test-product-p-987654",
		MatchStatus:     domain.MatchVerified,
		MatchMethod:     domain.MatchExplicit,
		Confidence:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
}

type gqlMockAdapter struct {
	fail bool
}

func (g *gqlMockAdapter) Source() domain.MarketplaceSource {
	return domain.MarketplaceTrendyol
}

func (g *gqlMockAdapter) DetectURL(rawURL string) (bool, string) {
	return true, "987654"
}

func (g *gqlMockAdapter) Fetch(_ context.Context, rawURL string) (*marketplace.FetchResult, error) {
	if g.fail {
		return nil, fmt.Errorf("upstream timeout")
	}
	price := any("555")
	return &marketplace.FetchResult{
		Source:          domain.MarketplaceTrendyol,
		SourceURL:       rawURL,
		SourceProductID: "987654",
		Product: &normalize.ProductInput{
			Name:         "GraphQL Sync Product",
			CurrentPrice: price,
			Rating:       4.5,
			ReviewCount:  3,
			StockStatus:  "in_stock",
		},
		FetchedAt: time.Now().UTC(),
	}, nil
}
