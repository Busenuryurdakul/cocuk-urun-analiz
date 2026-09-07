package cpsc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func testAdapter(srv *httptest.Server) *Adapter {
	u, err := url.Parse(srv.URL)
	if err != nil {
		panic(err)
	}
	return &Adapter{
		HTTP:         srv.Client(),
		BaseURL:      srv.URL,
		AllowedHosts: map[string]bool{u.Hostname(): true},
	}
}

const validFixture = `[
  {
    "RecallID": 123,
    "RecallNumber": "24-123",
    "RecallDate": "2024-01-15",
    "LastPublishDate": "2024-01-16T00:00:00",
    "Title": "Acme Baby Rattle Recall",
    "Description": "Choking hazard",
    "URL": "https://www.cpsc.gov/Recalls/2024/acme",
    "Unexpected": "ignored",
    "Products": [{"Name": "Baby Rattle", "Model": "X1"}],
    "Manufacturers": [{"Name": "Acme"}],
    "Hazards": [{"Name": "Choking"}],
    "Remedies": [{"Name": "Refund"}],
    "UPCs": [{"UPC": "012345678905"}]
  }
]`

func TestParseAndNormalize(t *testing.T) {
	items, err := ParseRecalls([]byte(validFixture))
	if err != nil || len(items) != 1 {
		t.Fatalf("parse: %v len=%d", err, len(items))
	}
	got := items[0]
	if got.Source != recall.SourceCPSC || got.SourceRecordID != "123" || got.Model != "X1" || got.UPC != "012345678905" {
		t.Fatalf("normalize: %+v", got)
	}
	if got.PublishedAt.IsZero() {
		t.Fatal("publishedAt")
	}
}

func TestParseEmptyAndInvalidDate(t *testing.T) {
	items, err := ParseRecalls([]byte("[]"))
	if err != nil || len(items) != 0 {
		t.Fatalf("empty: %v %d", err, len(items))
	}
	items, err = ParseRecalls([]byte(""))
	if err != nil || len(items) != 0 {
		t.Fatalf("blank: %v", err)
	}
	rec := Normalize(rawRecall{RecallID: "9", LastPublishDate: "not-a-date", RecallDate: "also-bad"})
	if !rec.PublishedAt.IsZero() {
		t.Fatalf("invalid date should be zero: %v", rec.PublishedAt)
	}
}

func TestSearchHTTPContract(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format=%s", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validFixture))
	}))
	defer srv.Close()

	ad := testAdapter(srv)
	items, err := ad.SearchRecalls(context.Background(), recall.Identity{Name: "rattle"})
	if err != nil || len(items) != 1 {
		t.Fatalf("search: %v %d", err, len(items))
	}

	empty, err := ad.SearchRecalls(context.Background(), recall.Identity{})
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty identity must not query all recalls: %v %d hits=%d", err, len(empty), hits)
	}
}

func TestSearchHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()
	ad := testAdapter(srv)
	if _, err := ad.SearchRecalls(context.Background(), recall.Identity{Name: "rattle"}); err == nil {
		t.Fatal("expected 500 error")
	}

	timeout := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(validFixture))
	}))
	defer timeout.Close()
	ad = testAdapter(timeout)
	ad.HTTP = timeout.Client()
	ad.HTTP.Timeout = 20 * time.Millisecond
	if _, err := ad.SearchRecalls(context.Background(), recall.Identity{Name: "rattle"}); err == nil {
		t.Fatal("expected timeout")
	}

	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("[" + strings.Repeat(`{"RecallID":1},`, 200000) + `{"RecallID":2}]`))
	}))
	defer big.Close()
	ad = testAdapter(big)
	if _, err := ad.SearchRecalls(context.Background(), recall.Identity{Name: "rattle"}); err == nil {
		t.Fatal("expected oversized rejection")
	}
}

func TestSearchRejectsUnknownHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(validFixture))
	}))
	defer srv.Close()
	ad := &Adapter{HTTP: srv.Client(), BaseURL: srv.URL}
	if _, err := ad.SearchRecalls(context.Background(), recall.Identity{Name: "rattle"}); err == nil {
		t.Fatal("unknown host must be rejected")
	}
}

func TestCPSCLiveSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_, err := NewAdapter().SearchRecalls(ctx, recall.Identity{Name: "Toddler"})
	if err != nil {
		t.Skipf("CPSC_LIVE_VERIFY IMPLEMENTED_BLOCKED: %v", err)
	}
}
