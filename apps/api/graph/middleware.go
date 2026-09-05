package graph

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/httpx"
)

func NewHandler(resolver *Resolver, authSvc *auth.Service) http.Handler {
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver}))
	return sessionMiddleware(authSvc, srv)
}

func sessionMiddleware(authSvc *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := httpx.WithRequest(r.Context(), r)
		ctx = httpx.WithResponseWriter(ctx, w)
		ctx = httpx.WithClientInfo(ctx, httpx.ClientInfoFromRequest(r))
		if access, ok := cookies.Get(r, cookies.AccessCookie); ok {
			if tokens, err := authSvc.ValidateAccessToken(ctx, access); err == nil {
				ctx = httpx.WithSession(ctx, httpx.SessionContext{
					AccessToken:    access,
					SessionID:      tokens.SessionID.Hex(),
					UserID:         tokens.UserID.Hex(),
					OrganizationID: tokens.OrganizationID.Hex(),
				})
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
