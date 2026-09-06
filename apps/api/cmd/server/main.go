package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/app"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	application.StartBackgroundWorkers(ctx)
	defer application.Shutdown(context.Background())

	health.SetReadinessCheck(application.Ready)

	resolver := &graph.Resolver{
		Auth:               application.Auth,
		Org:                application.Org,
		Consent:            application.Consent,
		Compliance:         application.Compliance,
		PolicyRepo:         application.PolicyRepo,
		ProductService:     application.Products,
		UGCService:         application.UGC,
		MarketplaceService: application.Marketplace,
		DatasetService:     application.Dataset,
		AgentService:       application.Agent,
		LLMService:         application.LLM,
		CookieOpts:         application.CookieOptions(),
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.WebBaseURL},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", health.Liveness)
	r.Get("/ready", health.Readiness)
	if application.AgentInternal != nil {
		application.AgentInternal.Register(r)
	}
	if application.LLMInternal != nil {
		application.LLMInternal.Register(r)
	}
	r.Handle("/graphql", graph.NewHandler(resolver, application.Auth))
	if cfg.AllowGraphQLPlayground {
		r.Handle("/", playground.Handler("Miyuna GraphQL", "/graphql"))
	}

	addr := cfg.Host + ":" + cfg.Port
	server := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("miyuna-api listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
