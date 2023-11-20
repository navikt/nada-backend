package api

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/quarto"
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

	storyHandler := story.NewHandler(repo)
	quartoHandler := quarto.NewHandler(repo, gcsClient, amplitudeClient, log.WithField("subsystem", "quarto"))

	router := chi.NewRouter()
	router.Use(corsMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.Handle("/query", authMW(gqlServer))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
		r.Put("/story", storyHandler.Update)
	})
	router.Route("/quarto/", func(r chi.Router) {
		r.Use(quartoHandler.QuartoMiddleware)
		r.Get("/*", quartoHandler.GetObject)
		r.Post("/create", quartoHandler.Create)
		r.Put("/update/{id}", quartoHandler.Update)
	})
	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
	})

	return router
}
