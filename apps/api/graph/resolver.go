package graph

import (
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/org"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/ugc"
)

type Resolver struct {
	Auth               *auth.Service
	Org                *org.Service
	Consent            *compliance.ConsentService
	Compliance         *compliance.Engine
	PolicyRepo         *compliance.PolicyRepository
	ProductService     *product.Service
	UGCService         *ugc.Service
	MarketplaceService *marketplace.Service
	DatasetService     *dataset.Service
	CookieOpts         cookies.Options
}
