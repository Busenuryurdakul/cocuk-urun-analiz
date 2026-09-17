package graph

import (
	"errors"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestToModelSyncPayloadRFC3339Semantics(t *testing.T) {
	now := time.Date(2025, 9, 17, 12, 30, 0, 0, time.UTC)
	prod := &domain.Product{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Name:           domain.ProductFieldMeta{Value: "Test"},
		CreatedAt:      now,
		UpdatedAt:      now,
		LastSyncedAt:   &now,
	}
	payload := toModelSyncPayload(&marketplace.SyncResult{
		Product:       prod,
		Updated:       true,
		UpdatedFields: []string{"price"},
		Source:        "TRENDYOL",
		SourceURL:     "https://www.trendyol.com/item-p-1",
		SyncedAt:      &now,
	})
	if payload.SyncedAt == nil {
		t.Fatal("expected syncedAt")
	}
	if _, err := time.Parse(time.RFC3339, *payload.SyncedAt); err != nil {
		t.Fatalf("syncedAt must be RFC3339: %v", err)
	}
	if payload.Product.LastSyncedAt == nil {
		t.Fatal("expected product.lastSyncedAt")
	}
	if _, err := time.Parse(time.RFC3339, *payload.Product.LastSyncedAt); err != nil {
		t.Fatalf("lastSyncedAt must be RFC3339: %v", err)
	}
}

func TestToModelSyncPayloadDeferredNullSyncedAt(t *testing.T) {
	now := time.Now().UTC()
	prod := &domain.Product{
		ID:        primitive.NewObjectID(),
		Name:      domain.ProductFieldMeta{Value: "Test"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	payload := toModelSyncPayload(&marketplace.SyncResult{
		Product: prod,
		Warning: marketplaceDeferredWarning(),
	})
	if payload.SyncedAt != nil {
		t.Fatal("deferred payload must have null syncedAt")
	}
	if payload.Updated {
		t.Fatal("expected updated=false")
	}
}

func TestMapPhase4ErrorLiveSyncCodes(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{marketplace.ErrNoSourceMapping, "NO_SOURCE_MAPPING"},
		{errors.Join(marketplace.ErrProviderFetch, errors.New("timeout")), "PROVIDER_FETCH_FAILED"},
		{marketplace.ErrForbidden, "FORBIDDEN"},
	}
	for _, tc := range cases {
		gqlErr := mapPhase4Error(tc.err)
		if gqlErr == nil {
			t.Fatalf("expected graphql error for %v", tc.err)
		}
		ext, ok := gqlErr.(*gqlerror.Error)
		if !ok {
			t.Fatalf("expected gqlerror.Error, got %T", gqlErr)
		}
		code, _ := ext.Extensions["code"].(string)
		if code != tc.wantCode {
			t.Fatalf("expected code %s, got %s", tc.wantCode, code)
		}
	}
}

func marketplaceDeferredWarning() string {
	return "Bu kaynak için canlı API erişimi yapılandırılmamış."
}
