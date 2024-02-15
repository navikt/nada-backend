package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/story"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
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
	teamCatalog teamkatalogen.Teamkatalogen,
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	gqlServer *handler.Server,
	promReg *prometheus.Registry,
	amplitudeClient amplitude.Amplitude,
	teamTokenCreds string,
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
		r.Get("/teamtokens", func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			authHeaderParts := strings.Split(authHeader, " ")
			if len(authHeaderParts) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if authHeaderParts[1] != teamTokenCreds {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenTeamMap, err := repo.GetNadaTokens(r.Context())
			if err != nil {
				log.WithError(err).Error("getting nada tokens")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			payloadBytes, err := json.Marshal(tokenTeamMap)
			if err != nil {
				log.WithError(err).Error("marshalling nada token map reponse")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payloadBytes)
		})
	})

	router.Route("/api/dataproducts", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			dpdto, apiErr := GetDataproduct(r.Context(), chi.URLParam(r, "id"))
			if apiErr != nil {
				apiErr.Log()
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
				apiErr.Log()
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

	router.Route("/api/productareas", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			padto, apiErr := GetProductAreas(r.Context())
			if apiErr != nil {
				apiErr.Log()
				http.Error(w, apiErr.Error(), apiErr.HttpStatus)
				return
			}
			err := json.NewEncoder(w).Encode(padto)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
	})

	return router
}
