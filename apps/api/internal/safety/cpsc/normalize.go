package cpsc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

type rawRecall struct {
	RecallID        any        `json:"RecallID"`
	RecallNumber    string     `json:"RecallNumber"`
	RecallDate      string     `json:"RecallDate"`
	Description     string     `json:"Description"`
	URL             string     `json:"URL"`
	Title           string     `json:"Title"`
	LastPublishDate string     `json:"LastPublishDate"`
	Products        []rawNamed `json:"Products"`
	Manufacturers   []rawNamed `json:"Manufacturers"`
	Hazards         []rawNamed `json:"Hazards"`
	Remedies        []rawNamed `json:"Remedies"`
	UPCs            []rawUPC   `json:"UPCs"`
}

type rawNamed struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Model       string `json:"Model"`
	Type        string `json:"Type"`
	CompanyID   string `json:"CompanyID"`
}

type rawUPC struct {
	UPC string `json:"UPC"`
}

func ParseRecalls(body []byte) ([]recall.Record, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return []recall.Record{}, nil
	}
	var items []rawRecall
	if err := json.Unmarshal(body, &items); err != nil {
		var single rawRecall
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("cpsc schema: %w", err)
		}
		items = []rawRecall{single}
	}
	out := make([]recall.Record, 0, len(items))
	for _, item := range items {
		out = append(out, Normalize(item))
	}
	return out, nil
}

func Normalize(item rawRecall) recall.Record {
	productName, model := firstProduct(item.Products)
	manufacturer := firstName(item.Manufacturers)
	hazard := firstName(item.Hazards)
	remedy := firstName(item.Remedies)
	upc := firstUPC(item.UPCs)
	id := stringifyID(item.RecallID)
	if id == "" {
		id = strings.TrimSpace(item.RecallNumber)
	}
	published := parseCPSCDate(item.LastPublishDate)
	if published.IsZero() {
		published = parseCPSCDate(item.RecallDate)
	}
	return recall.Record{
		Source:         recall.SourceCPSC,
		SourceRecordID: normalize.NormalizeIdentifier(id),
		Title:          strings.TrimSpace(item.Title),
		Brand:          manufacturer,
		ProductName:    productName,
		Model:          normalize.NormalizeIdentifier(model),
		Manufacturer:   manufacturer,
		UPC:            normalize.NormalizeIdentifier(upc),
		EAN:            "",
		GTIN:           "",
		Hazard:         hazard,
		Remedy:         remedy,
		Description:    strings.TrimSpace(item.Description),
		PublishedAt:    published,
		Reference:      strings.TrimSpace(item.URL),
		RawMetadata: map[string]any{
			"recallNumber": item.RecallNumber,
			"recallId":     id,
		},
	}
}

func firstProduct(items []rawNamed) (name, model string) {
	if len(items) == 0 {
		return "", ""
	}
	return strings.TrimSpace(items[0].Name), strings.TrimSpace(items[0].Model)
}

func firstName(items []rawNamed) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[0].Name)
}

func firstUPC(items []rawUPC) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[0].UPC)
}

func stringifyID(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func parseCPSCDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02", "1/2/2006", "01/02/2006"}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}
