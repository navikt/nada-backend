package api

import (
	"embed"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

//go:embed swagger/*
var swagger embed.FS

func NewRouter(repo *database.Repo, oauth2Config oauth2.Config, log *logrus.Entry, projectsMapping *auth.TeamProjectsUpdater, gcp GCP, middlewares ...openapi.MiddlewareFunc) http.Handler {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	srv := New(repo, oauth2Config, log.WithField("subsystem", "api"), projectsMapping, gcp)

	latencyHistBuckets := []float64{.001, .005, .01, .025, .05, .1, .5, 1, 3, 5}
	prometheusMiddleware := PrometheusMiddleware("backend", latencyHistBuckets...)
	prometheusMiddleware.Initialize("/api/v1/", http.MethodGet, http.StatusOK)

	baseRouter := chi.NewRouter()
	baseRouter.Use(prometheusMiddleware.Handler())
	baseRouter.Use(corsMW)
	baseRouter.Get("/api/login", srv.Login)
	baseRouter.Get("/api/oauth2/callback", srv.Callback)
	baseRouter.Get("/internal/isalive", func(rw http.ResponseWriter, r *http.Request) {})
	baseRouter.Get("/internal/isready", func(rw http.ResponseWriter, r *http.Request) {})
	baseRouter.Get("/internal/metrics", promhttp.Handler().(http.HandlerFunc))
	baseRouter.Get("/api/spec", func(rw http.ResponseWriter, r *http.Request) {
		spec, err := openapi.GetSwagger()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(rw).Encode(spec)
	})

	baseRouter.Handle("/api/*", http.StripPrefix("/api/", http.FileServer(http.FS(swagger))))

	router := openapi.HandlerWithOptions(srv, openapi.ChiServerOptions{BaseRouter: baseRouter, BaseURL: "/api", Middlewares: middlewares})
	return router
}
