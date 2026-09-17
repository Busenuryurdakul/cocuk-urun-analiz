package marketplace

import (
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
)

// ProviderSettings holds authorized marketplace API credentials (server-side only).
type ProviderSettings struct {
	FetchTimeout          time.Duration
	TrendyolAPIBaseURL    string
	TrendyolAPIKey        string
	TrendyolAPISecret     string
	TrendyolSupplierID    string
	HepsiburadaAPIBaseURL string
	HepsiburadaAPIKey     string
	HepsiburadaAPISecret  string
}

// SettingsFromConfig maps application config to marketplace provider settings.
func SettingsFromConfig(cfg config.Config) ProviderSettings {
	return ProviderSettings{
		FetchTimeout:          cfg.MarketplaceFetchTimeout,
		TrendyolAPIBaseURL:    strings.TrimSpace(cfg.TrendyolAPIBaseURL),
		TrendyolAPIKey:        strings.TrimSpace(cfg.TrendyolAPIKey),
		TrendyolAPISecret:     strings.TrimSpace(cfg.TrendyolAPISecret),
		TrendyolSupplierID:    strings.TrimSpace(cfg.TrendyolSupplierID),
		HepsiburadaAPIBaseURL: strings.TrimSpace(cfg.HepsiburadaAPIBaseURL),
		HepsiburadaAPIKey:     strings.TrimSpace(cfg.HepsiburadaAPIKey),
		HepsiburadaAPISecret:  strings.TrimSpace(cfg.HepsiburadaAPISecret),
	}
}

func (s ProviderSettings) TrendyolConfigured() bool {
	return s.TrendyolAPIBaseURL != "" &&
		s.TrendyolAPIKey != "" &&
		s.TrendyolAPISecret != "" &&
		s.TrendyolSupplierID != ""
}

func (s ProviderSettings) HepsiburadaConfigured() bool {
	return s.HepsiburadaAPIBaseURL != "" &&
		s.HepsiburadaAPIKey != "" &&
		s.HepsiburadaAPISecret != ""
}
