package api

import (
	"encoding/json"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/story"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type HTTPAPI interface {
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

func New(
	repo *database.Repo,
	gcsClient *gcs.Client,
	teamCatalog graph.Teamkatalogen,
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	gqlServer *handler.Server,
	promReg *prometheus.Registry,
	amplitudeClient amplitude.Amplitude,
	log *logrus.Logger,
) *chi.Mux {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	storyHandler := story.NewHandler(repo, gcsClient, teamCatalog, amplitudeClient, log.WithField("subsystem", "story"))

	router := chi.NewRouter()
	router.Use(corsMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.Handle("/query", authMW(gqlServer))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
	})
	router.Route(`/{story|quarto}/`, func(r chi.Router) {
		r.Use(storyHandler.Middleware)
		r.Get("/*", storyHandler.GetObject)
		r.Post("/create", storyHandler.Create)
		r.Route("/update/{id}", func(r chi.Router) {
			r.Put("/", storyHandler.Update)
			r.Patch("/", storyHandler.Append)
		})
	})
	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	})

	router.Route("/api/dataproducts", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			dpdto, apiErr := GetDataproduct(r.Context(), chi.URLParam(r, "id"))
			if apiErr != nil {
				http.Error(w, apiErr.Error(), apiErr.HttpStatus)
				return
			}
			err := json.NewEncoder(w).Encode(dpdto)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
	})

	router.Route("/api/datasets", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			dsdto, apiErr := GetDataset(r.Context(), chi.URLParam(r, "id"))
			if apiErr != nil {
				http.Error(w, apiErr.Error(), apiErr.HttpStatus)
				return
			}
			err := json.NewEncoder(w).Encode(dsdto)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
	})

	return router
}
