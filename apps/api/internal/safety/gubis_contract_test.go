package safety_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/gubis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func TestGUBISNormalizeAndEmpty(t *testing.T) {
	rec, err := gubis.Normalize(gubis.Notice{
		ID: "g-1", Title: "Güvensiz oyuncak", Brand: "Acme", ProductName: "Çıngırak",
		Model: "X1", Risk: "Boğulma", Action: "Toplatma", PublishedAt: "2024-02-01",
		Reference: "https://gubis.ticaret.gov.tr/Bildirim/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Source != "GUBIS" || rec.SourceRecordID != "G-1" || rec.Hazard != "Boğulma" {
		t.Fatalf("%+v", rec)
	}
	if _, err := gubis.Normalize(gubis.Notice{}); err == nil {
		t.Fatal("missing id")
	}
	rec, err = gubis.Normalize(gubis.Notice{ID: "g-2", PublishedAt: "bad-date"})
	if err != nil || !rec.PublishedAt.IsZero() {
		t.Fatalf("invalid date: %v %+v", err, rec)
	}
}

func TestGUBISNormalizeMissingOptionalFields(t *testing.T) {
	rec, err := gubis.Normalize(gubis.Notice{ID: "g-3"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Brand != "" || rec.ProductName != "" || rec.Hazard != "" || !rec.PublishedAt.IsZero() {
		t.Fatalf("optional fields should stay empty: %+v", rec)
	}
}

func TestGUBISSearchHasNoStableAPI(t *testing.T) {
	_, err := gubis.NewAdapter().SearchRecalls(context.Background(), recall.Identity{Name: "çıngırak"})
	if !errors.Is(err, gubis.ErrNoStablePublicAPI) {
		t.Fatalf("expected no stable API, got %v", err)
	}
}
