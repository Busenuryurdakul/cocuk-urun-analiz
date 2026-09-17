package marketplace

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxProviderResponseBytes = 2 << 20 // 2 MiB

func newProviderHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func providerGET(ctx context.Context, client *http.Client, baseURL, path, basicUser, basicPass string) ([]byte, int, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("provider base url not configured")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, 0, fmt.Errorf("provider path required")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	reqURL := base + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	if basicUser != "" || basicPass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(basicUser + ":" + basicPass))
		req.Header.Set("Authorization", "Basic "+token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Miyuna/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
