package gubis

import (
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

type Notice struct {
	ID           string
	Title        string
	Brand        string
	ProductName  string
	Model        string
	Risk         string
	Action       string
	PublishedAt  string
	Reference    string
	Manufacturer string
	GTIN         string
	UPC          string
	EAN          string
}

func Normalize(notice Notice) (recall.Record, error) {
	id := strings.TrimSpace(notice.ID)
	if id == "" {
		return recall.Record{}, fmt.Errorf("gubis notice missing sourceRecordId")
	}
	published := parseDate(notice.PublishedAt)
	return recall.Record{
		Source:         recall.SourceGUBIS,
		SourceRecordID: normalize.NormalizeIdentifier(id),
		Title:          strings.TrimSpace(notice.Title),
		Brand:          strings.TrimSpace(notice.Brand),
		ProductName:    strings.TrimSpace(notice.ProductName),
		Model:          normalize.NormalizeIdentifier(notice.Model),
		Manufacturer:   strings.TrimSpace(notice.Manufacturer),
		GTIN:           normalize.NormalizeIdentifier(notice.GTIN),
		UPC:            normalize.NormalizeIdentifier(notice.UPC),
		EAN:            normalize.NormalizeIdentifier(notice.EAN),
		Hazard:         strings.TrimSpace(notice.Risk),
		Remedy:         strings.TrimSpace(notice.Action),
		PublishedAt:    published,
		Reference:      strings.TrimSpace(notice.Reference),
		RawMetadata: map[string]any{
			"source": "GUBIS",
		},
	}, nil
}

func parseDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "02.01.2006"} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}
