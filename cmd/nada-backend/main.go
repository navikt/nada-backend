package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/middleware"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var cfg = DefaultConfig()

const (
	TeamsUpdateFrequency        = 5 * time.Minute
	EnsureAccessUpdateFrequency = 5 * time.Minute
	TeamProjectsUpdateFrequency = 5 * time.Minute
)

func init() {
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.DBConnectionDSN, "db-connection-dsn", fmt.Sprintf("%v?sslmode=disable", getEnv("NAIS_DATABASE_NADA_BACKEND_NADA_URL", "postgres://postgres:postgres@127.0.0.1:5432/nada")), "database connection DSN")
	flag.StringVar(&cfg.OAuth2.ClientID, "oauth2-client-id", os.Getenv("AZURE_APP_CLIENT_ID"), "OAuth2 client ID")
	flag.StringVar(&cfg.OAuth2.ClientSecret, "oauth2-client-secret", os.Getenv("AZURE_APP_CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&cfg.OAuth2.TenantID, "oauth2-tenant-id", os.Getenv("AZURE_APP_TENANT_ID"), "Azure tenant id")
	flag.StringVar(&cfg.Hostname, "hostname", os.Getenv("HOSTNAME"), "Hostname the application is served from")
	flag.StringVar(&cfg.TeamsURL, "teams-url", cfg.TeamsURL, "URL for json containing teams and UUIDs")
	flag.StringVar(&cfg.ProdTeamProjectsOutputURL, "prod-team-projects-url", cfg.ProdTeamProjectsOutputURL, "URL for json containing prod team projects")
	flag.StringVar(&cfg.DevTeamProjectsOutputURL, "dev-team-projects-url", cfg.DevTeamProjectsOutputURL, "URL for json containing dev team projects")
	flag.StringVar(&cfg.TeamsToken, "teams-token", os.Getenv("GITHUB_READ_TOKEN"), "Token for accessing teams json")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.CookieSecret, "cookie-secret", "", "Secret used when encrypting cookies")
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)

	repo, err := database.New(cfg.DBConnectionDSN)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	teamUUIDs := make(map[string]string)
	go auth.UpdateTeams(ctx, teamUUIDs, cfg.TeamsURL, cfg.TeamsToken, TeamsUpdateFrequency)

	teamProjectsMapping := make(map[string][]string)
	go teamprojectsupdater.New(ctx, teamProjectsMapping, cfg.DevTeamProjectsOutputURL, cfg.ProdTeamProjectsOutputURL, cfg.TeamsToken, TeamProjectsUpdateFrequency, nil).Run()

	//iam := iam.New(ctx)
	//go accessensurer.New(ctx, cfg, firestore, iam, EnsureAccessUpdateFrequency).Run()

	azureGroups := auth.NewAzureGroups(http.DefaultClient, cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID)
	authenticatorMiddleware := middleware.JWTValidatorMiddleware(auth.KeyDiscoveryURL(cfg.OAuth2.TenantID), cfg.OAuth2.ClientID, false, azureGroups, teamUUIDs)
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	oauth2Config := auth.CreateOAuth2Config(cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID, cfg.Hostname)
	srv := api.New(repo, oauth2Config, log.WithField("subsystem", "api"))

	baseRouter := chi.NewRouter()
	baseRouter.Use(corsMW)
	baseRouter.Get("/api/login", srv.Login)
	baseRouter.Get("/api/oauth2/callback", srv.Callback)

	router := openapi.HandlerWithOptions(srv, openapi.ChiServerOptions{BaseRouter: baseRouter, BaseURL: "/api", Middlewares: []openapi.MiddlewareFunc{authenticatorMiddleware}})
	log.Info("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getEnv(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return fallback
}
