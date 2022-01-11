package api

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
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
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	gqlServer *handler.Server,
	promReg *prometheus.Registry,
	log *logrus.Logger,
) *chi.Mux {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	datapackageRedirect := func(w http.ResponseWriter, r *http.Request) {
		host := "https://datapakker.dev.intern.nav.no"
		if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
			host = "https://datapakker.intern.nav.no"
		}

		http.Redirect(w, r, host+r.URL.Path, http.StatusPermanentRedirect)
	}
	storyHandler := story.NewHandler(repo)

	router := chi.NewRouter()
	router.Use(corsMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.Handle("/query", authMW(gqlServer))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
		r.HandleFunc("/nav-interndata/*", datapackageRedirect)
		r.Post("/story", storyHandler.Upload)
		r.Put("/story/{id}", storyHandler.Update)
	})
	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	})
	router.HandleFunc("/datapakke/*", datapackageRedirect)

	return router
}
