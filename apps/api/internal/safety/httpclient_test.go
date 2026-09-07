package safety

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoOfficialGETRejectsUnknownHost(t *testing.T) {
	_, _, err := DoOfficialGET(context.Background(), http.DefaultClient, "https://evil.example/recall", map[string]bool{"www.saferproducts.gov": true})
	if err == nil {
		t.Fatal("expected host rejection")
	}
}

func TestReadBoundedBodyRejectsOversized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxOfficialResponseBytes+8)))
	}))
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBoundedBody(resp); err == nil {
		t.Fatal("expected oversized rejection")
	}
}

func TestOfficialClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 15 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, _, err := DoOfficialGET(ctx, client, srv.URL, map[string]bool{"127.0.0.1": true, "localhost": true}); err == nil {
		t.Fatal("expected timeout or cancel")
	}
}
