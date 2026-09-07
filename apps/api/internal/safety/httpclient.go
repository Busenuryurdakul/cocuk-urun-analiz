package safety

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/fetch"
)

const (
	MaxOfficialResponseBytes = 2 << 20
	OfficialRequestTimeout   = 8 * time.Second
	OfficialMaxRetries       = 2
)

func NewOfficialHTTPClient(allowedHosts map[string]bool) *http.Client {
	return &http.Client{
		Timeout: OfficialRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= fetch.MaxRedirects {
				return fetch.ErrTooManyRedirects
			}
			hops := make([]string, 0, len(via))
			for _, v := range via {
				hops = append(hops, v.URL.String())
			}
			original := hops[0]
			if req.URL != nil {
				hops = append(hops, req.URL.String())
			}
			return fetch.ValidateRedirectChain(original, hops[1:], allowedHosts)
		},
	}
}

func ReadBoundedBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("empty response")
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, MaxOfficialResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > MaxOfficialResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", MaxOfficialResponseBytes)
	}
	return body, nil
}

func ValidateOfficialURL(raw string, allowedHosts map[string]bool) error {
	_, err := fetch.ValidateURLStructure(raw, allowedHosts)
	return err
}

func DoOfficialGET(ctx context.Context, client *http.Client, rawURL string, allowedHosts map[string]bool) (*http.Response, []byte, error) {
	if err := ValidateOfficialURL(rawURL, allowedHosts); err != nil {
		return nil, nil, err
	}
	var lastErr error
	for attempt := 0; attempt <= OfficialMaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "miyuna-safety-adapter/1.0")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := ReadBoundedBody(resp)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 500 && attempt < OfficialMaxRetries {
			lastErr = fmt.Errorf("official source status %d", resp.StatusCode)
			continue
		}
		return resp, body, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("official source request failed")
	}
	return nil, nil, lastErr
}
