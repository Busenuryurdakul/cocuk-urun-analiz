package fetch

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidURL       = errors.New("invalid url")
	ErrHostNotAllowed   = errors.New("host not allowed")
	ErrPrivateIPBlocked = errors.New("private ip blocked")
	ErrTooManyRedirects = errors.New("too many redirects")
	ErrSchemeNotAllowed = errors.New("scheme not allowed")
)

const (
	MaxRedirects = 3
	MaxURLLength = 2048
)

var defaultAllowedHosts = map[string]bool{
	"hepsiburada.com":     true,
	"www.hepsiburada.com": true,
	"trendyol.com":        true,
	"www.trendyol.com":    true,
}

// ValidateURL checks scheme, length, host allowlist, and blocks private/reserved IPs.
func ValidateURL(raw string, allowedHosts map[string]bool) (*url.URL, error) {
	parsed, err := ValidateURLStructure(raw, allowedHosts)
	if err != nil {
		return nil, err
	}
	if err := validateHostResolution(parsed.Hostname()); err != nil {
		return nil, err
	}
	return parsed, nil
}

// ValidateURLStructure validates URL shape and host allowlist without DNS resolution.
func ValidateURLStructure(raw string, allowedHosts map[string]bool) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidURL)
	}
	if len(raw) > MaxURLLength {
		return nil, fmt.Errorf("%w: too long", ErrInvalidURL)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrSchemeNotAllowed
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	host := strings.ToLower(parsed.Hostname())
	if allowedHosts == nil {
		allowedHosts = defaultAllowedHosts
	}
	if !hostAllowed(host, allowedHosts) {
		return nil, ErrHostNotAllowed
	}
	return parsed, nil
}

func validateHostResolution(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return ErrPrivateIPBlocked
		}
		return nil
	}
	return blockPrivateIP(host)
}

func hostAllowed(host string, allowed map[string]bool) bool {
	if allowed[host] {
		return true
	}
	for allowedHost := range allowed {
		if strings.HasSuffix(host, "."+allowedHost) {
			return true
		}
	}
	return false
}

func blockPrivateIP(host string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS failure is not SSRF success path; treat as blocked for safety.
		return fmt.Errorf("%w: dns lookup failed", ErrPrivateIPBlocked)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return ErrPrivateIPBlocked
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	// Block metadata endpoints and CGNAT ranges commonly abused in SSRF.
	blockedCIDRs := []string{
		"169.254.0.0/16",
		"100.64.0.0/10",
		"0.0.0.0/8",
	}
	for _, cidr := range blockedCIDRs {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateRedirectChain ensures redirect count stays within limit and each hop is allowed.
func ValidateRedirectChain(original string, hops []string, allowedHosts map[string]bool) error {
	if len(hops) > MaxRedirects {
		return ErrTooManyRedirects
	}
	if _, err := ValidateURLStructure(original, allowedHosts); err != nil {
		return err
	}
	for _, hop := range hops {
		if _, err := ValidateURLStructure(hop, allowedHosts); err != nil {
			return err
		}
	}
	return nil
}
