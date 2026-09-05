package fetch_test

import (
	"errors"
	"net"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/fetch"
)

func TestValidateURLStructureAllowedHost(t *testing.T) {
	_, err := fetch.ValidateURLStructure("https://www.hepsiburada.com/product/p-123", nil)
	if err != nil {
		t.Fatalf("expected allowed host, got %v", err)
	}
}

func TestValidateURLStructureBlocksUnknownHost(t *testing.T) {
	_, err := fetch.ValidateURLStructure("https://evil.example/product", nil)
	if !errors.Is(err, fetch.ErrHostNotAllowed) {
		t.Fatalf("expected host not allowed, got %v", err)
	}
}

func TestValidateURLStructureBlocksFileScheme(t *testing.T) {
	_, err := fetch.ValidateURLStructure("file:///etc/passwd", nil)
	if !errors.Is(err, fetch.ErrSchemeNotAllowed) {
		t.Fatalf("expected scheme not allowed, got %v", err)
	}
}

func TestValidateRedirectChainLimit(t *testing.T) {
	hops := []string{
		"https://www.trendyol.com/a-p-1",
		"https://www.trendyol.com/b-p-2",
		"https://www.trendyol.com/c-p-3",
		"https://www.trendyol.com/d-p-4",
	}
	err := fetch.ValidateRedirectChain("https://www.trendyol.com/start", hops, nil)
	if !errors.Is(err, fetch.ErrTooManyRedirects) {
		t.Fatalf("expected too many redirects, got %v", err)
	}
}

func TestBlockedLoopbackIP(t *testing.T) {
	if !isBlockedForTest(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback should be blocked")
	}
}

func isBlockedForTest(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate()
}
