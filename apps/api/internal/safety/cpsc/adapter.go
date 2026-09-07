package cpsc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

const DefaultBaseURL = "https://www.saferproducts.gov/RestWebServices/Recall"

var AllowedHosts = map[string]bool{
	"www.saferproducts.gov": true,
	"saferproducts.gov":     true,
}

type Adapter struct {
	HTTP         *http.Client
	BaseURL      string
	AllowedHosts map[string]bool
}

func NewAdapter() *Adapter {
	return &Adapter{
		HTTP:         safety.NewOfficialHTTPClient(AllowedHosts),
		BaseURL:      DefaultBaseURL,
		AllowedHosts: AllowedHosts,
	}
}

func (a *Adapter) hosts() map[string]bool {
	if len(a.AllowedHosts) > 0 {
		return a.AllowedHosts
	}
	return AllowedHosts
}

func (a *Adapter) Source() string { return recall.SourceCPSC }

func (a *Adapter) SearchRecalls(ctx context.Context, identity recall.Identity) ([]recall.Record, error) {
	if a.HTTP == nil {
		a.HTTP = safety.NewOfficialHTTPClient(a.hosts())
	}
	base := a.BaseURL
	if strings.TrimSpace(base) == "" {
		base = DefaultBaseURL
	}
	query := url.Values{}
	query.Set("format", "json")
	if identity.UPC != "" {
		query.Set("UPC", identity.UPC)
	} else if identity.Name != "" {
		query.Set("ProductName", identity.Name)
	} else if identity.Manufacturer != "" {
		query.Set("Manufacturer", identity.Manufacturer)
	} else if identity.Brand != "" {
		query.Set("Manufacturer", identity.Brand)
	} else {
		return []recall.Record{}, nil
	}
	rawURL := base + "?" + query.Encode()
	resp, body, err := safety.DoOfficialGET(ctx, a.HTTP, rawURL, a.hosts())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cpsc status %d", resp.StatusCode)
	}
	return ParseRecalls(body)
}
