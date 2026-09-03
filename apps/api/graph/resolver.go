package graph

import (
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
)

type Resolver struct {
	Auth       *auth.Service
	CookieOpts cookies.Options
}
