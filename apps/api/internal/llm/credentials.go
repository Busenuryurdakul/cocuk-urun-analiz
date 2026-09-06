package llm

import (
	"net"
	"net/url"
	"strings"
)

func ProviderCredentialsConfigured(baseURL, apiKey string) bool {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return false
	}
	if strings.TrimSpace(apiKey) != "" {
		return true
	}
	return ProviderAllowsEmptyAPIKey(baseURL)
}

func ProviderAllowsEmptyAPIKey(baseURL string) bool {
	raw := strings.TrimSpace(baseURL)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return false
	}
	if host == "localhost" || host == "host.docker.internal" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	return strings.HasSuffix(host, ".local")
}
