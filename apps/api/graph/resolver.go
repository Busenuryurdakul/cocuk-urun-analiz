package graph

import (
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/org"
)

type Resolver struct {
	Auth       *auth.Service
	Org        *org.Service
	Consent    *compliance.ConsentService
	Compliance *compliance.Engine
	PolicyRepo *compliance.PolicyRepository
	CookieOpts cookies.Options
}
