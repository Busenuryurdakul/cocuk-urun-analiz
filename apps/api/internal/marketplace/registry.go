package marketplace

import (
	"net/url"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	return &Registry{adapters: adapters}
}

func DefaultRegistry() *Registry {
	return NewRegistry(NewHepsiburadaAdapter(), NewTrendyolAdapter())
}

func (r *Registry) Detect(rawURL string) (domain.MarketplaceSource, string, bool) {
	for _, adapter := range r.adapters {
		if ok, productID := adapter.DetectURL(rawURL); ok {
			return adapter.Source(), productID, true
		}
	}
	return "", "", false
}

func (r *Registry) Get(source domain.MarketplaceSource) (Adapter, bool) {
	for _, adapter := range r.adapters {
		if adapter.Source() == source {
			return adapter, true
		}
	}
	return nil, false
}

func hostMatches(u *url.URL, suffix string) bool {
	host := strings.ToLower(u.Hostname())
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}
