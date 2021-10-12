package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/metadata"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
)

var cfg = DefaultConfig()

const (
	TeamsUpdateFrequency           = 5 * time.Minute
	EnsureAccessUpdateFrequency    = 5 * time.Minute
	TeamProjectsUpdateFrequency    = 5 * time.Minute
	DatasetMetadataUpdateFrequency = 5 * time.Minute
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
	flag.BoolVar(&cfg.MockAuth, "mock-auth", false, "Use mock authentication")
	flag.BoolVar(&cfg.SkipMetadataSync, "skip-metadata-sync", false, "Skip metadata sync")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log := newLogger()

	repo, err := database.New(cfg.DBConnectionDSN)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	authenticatorMiddleware := auth.MockJWTValidatorMiddleware()
	teamProjectsMapping := &auth.MockTeamProjectsUpdater
	oauth2Config := oauth2.Config{}
	if !cfg.MockAuth {
		teamsCache := auth.NewTeamsCache(cfg.TeamsURL, cfg.TeamsToken)
		go teamsCache.Run(ctx, TeamsUpdateFrequency)

		teamProjectsMapping = auth.NewTeamProjectsUpdater(cfg.DevTeamProjectsOutputURL, cfg.ProdTeamProjectsOutputURL, cfg.TeamsToken, http.DefaultClient)
		go teamProjectsMapping.Run(ctx, TeamProjectsUpdateFrequency)

		azure := auth.NewAzure(cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID, cfg.Hostname)
		authenticatorMiddleware = azure.Middleware(teamsCache)
		oauth2Config = azure.OAuth2Config()
	}

	var gcp api.GCP
	if !cfg.SkipMetadataSync {
		datacatalogClient, err := metadata.NewDatacatalog(ctx)
		if err != nil {
			log.WithError(err).Fatal("creating datacatalog client")
		}
		gcp = datacatalogClient
		datasetEnricher := metadata.New(datacatalogClient, repo, log.WithField("subsystem", "datasetenricher"))
		go datasetEnricher.Run(ctx, DatasetMetadataUpdateFrequency)
	}

	router := api.NewRouter(repo, oauth2Config, log.WithField("subsystem", "api"), teamProjectsMapping, gcp, authenticatorMiddleware)
	log.Info("Listening on :8080")

	server := http.Server{
		Addr:    cfg.BindAddress,
		Handler: router,
	}
	go server.ListenAndServe()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Warn("Shutdown error")
	}
}

func newLogger() *logrus.Logger {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
	return log
}

func getEnv(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return fallback
}
