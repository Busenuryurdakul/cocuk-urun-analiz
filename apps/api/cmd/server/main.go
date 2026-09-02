package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Minimal REST — health/readiness only (no public business REST API)
	r.Get("/health", health.Liveness)
	r.Get("/ready", health.Readiness)

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	r.Handle("/graphql", srv)
	if cfg.AllowGraphQLPlayground {
		r.Handle("/", playground.Handler("Miyuna GraphQL", "/graphql"))
	}

	addr := cfg.Host + ":" + cfg.Port
	log.Printf("miyuna-api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
