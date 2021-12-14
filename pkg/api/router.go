package api

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/story"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func New(repo *database.Repo, gcp graph.Bigquery, oauth2 OAuth2, gcpProjects *auth.TeamProjectsUpdater, accessMgr graph.AccessManager, authMW auth.MiddlewareHandler, tk *teamkatalogen.Teamkatalogen, promReg *prometheus.Registry, log *logrus.Logger) *chi.Mux {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	httpAPI := new(oauth2, log.WithField("subsystem", "api"))

	gqlServer := graph.New(repo, gcp, gcpProjects, accessMgr, tk, log.WithField("subsystem", "graph"))

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
	})
	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	})
	router.HandleFunc("/datapakke/*", datapackageRedirect)

	return router
}
