package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/metabase"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/iam/v1"
)

var (
	cfg = DefaultConfig()

	promErrs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "nada_backend",
		Name:      "errors",
	}, []string{"location"})
)

const (
	TeamProjectsUpdateFrequency    = 5 * time.Minute
	DatasetMetadataUpdateFrequency = 1 * time.Hour
	AccessEnsurerFrequency         = 5 * time.Minute
	MetabaseUpdateFrequency        = 5 * time.Minute
)

func init() {
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.DBConnectionDSN, "db-connection-dsn", fmt.Sprintf("%v?sslmode=disable", getEnv("NAIS_DATABASE_NADA_BACKEND_NADA_URL", "postgres://postgres:postgres@127.0.0.1:5432/nada")), "database connection DSN")
	flag.StringVar(&cfg.Hostname, "hostname", os.Getenv("HOSTNAME"), "Hostname the application is served from")
	flag.StringVar(&cfg.OAuth2.ClientID, "oauth2-client-id", os.Getenv("CLIENT_ID"), "OAuth2 client ID")
	flag.StringVar(&cfg.OAuth2.ClientSecret, "oauth2-client-secret", os.Getenv("CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&cfg.ProdTeamProjectsOutputURL, "prod-team-projects-url", cfg.ProdTeamProjectsOutputURL, "URL for json containing prod team projects")
	flag.StringVar(&cfg.DevTeamProjectsOutputURL, "dev-team-projects-url", cfg.DevTeamProjectsOutputURL, "URL for json containing dev team projects")
	flag.StringVar(&cfg.TeamsToken, "teams-token", os.Getenv("GITHUB_READ_TOKEN"), "Token for accessing teams json")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.CookieSecret, "cookie-secret", "", "Secret used when encrypting cookies")
	flag.StringVar(&cfg.TeamkatalogenURL, "teamkatalogen-url", cfg.TeamkatalogenURL, "Teamkatalog API URL")
	flag.BoolVar(&cfg.MockAuth, "mock-auth", false, "Use mock authentication")
	flag.StringVar(&cfg.GoogleAdminImpersonationSubject, "google-admin-subject", os.Getenv("GOOGLE_ADMIN_IMPERSONATION_SUBJECT"), "Subject to impersonate when accessing google admin apis")
	flag.StringVar(&cfg.ServiceAccountFile, "service-account-file", os.Getenv("GOOGLE_ADMIN_CREDENTIALS_PATH"), "Service account file for accessing google admin apis")
	flag.BoolVar(&cfg.SkipMetadataSync, "skip-metadata-sync", false, "Skip metadata sync")
	flag.StringVar(&cfg.MetabaseServiceAccountFile, "metabase-service-account-file", os.Getenv("METABASE_GOOGLE_CREDENTIALS_PATH"), "Service account file for metabase access to bigquery tables")
	flag.StringVar(&cfg.MetabaseUsername, "metabase-username", os.Getenv("METABASE_USERNAME"), "Username for metabase api")
	flag.StringVar(&cfg.MetabasePassword, "metabase-password", os.Getenv("METABASE_PASSWORD"), "Password for metabase api")
	flag.StringVar(&cfg.MetabaseAPI, "metabase-api", os.Getenv("METABASE_API"), "URL to Metabase API, including scheme and `/api`")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log := newLogger()

	repo, err := database.New(cfg.DBConnectionDSN, log.WithField("subsystem", "repo"))
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	authenticatorMiddleware := auth.MockJWTValidatorMiddleware()
	teamProjectsMapping := &auth.MockTeamProjectsUpdater
	var oauth2Config api.OAuth2
	var accessMgr graph.AccessManager
	accessMgr = access.NewNoop()
	if !cfg.MockAuth {
		teamProjectsMapping = auth.NewTeamProjectsUpdater(cfg.DevTeamProjectsOutputURL, cfg.ProdTeamProjectsOutputURL, cfg.TeamsToken, http.DefaultClient)
		go teamProjectsMapping.Run(ctx, TeamProjectsUpdateFrequency)

		googleGroups, err := bigquery.NewGoogleGroups(ctx, cfg.ServiceAccountFile, cfg.GoogleAdminImpersonationSubject, log.WithField("subsystem", "googlegroups"))
		if err != nil {
			log.Fatal(err)
		}

		gauth := auth.NewGoogle(cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.Hostname)
		oauth2Config = gauth
		authenticatorMiddleware = gauth.Middleware(googleGroups)
		accessMgr = access.NewBigquery()
	}

	if err := runMetabase(ctx, log.WithField("subsystem", "metabase"), cfg, repo, accessMgr); err != nil {
		log.WithError(err).Fatal("running metabase")
	}

	go access.NewEnsurer(repo, accessMgr, promErrs, log.WithField("subsystem", "accessensurer")).Run(ctx, AccessEnsurerFrequency)

	var gcp graph.Bigquery
	if !cfg.SkipMetadataSync {
		datacatalogClient, err := bigquery.New(ctx)
		if err != nil {
			log.WithError(err).Fatal("Creating datacatalog client")
		}

		gcp = datacatalogClient
		de := bigquery.NewDatasetEnricher(datacatalogClient, repo, log.WithField("subsystem", "datasetenricher"))
		go de.Run(ctx, DatasetMetadataUpdateFrequency)
	}

	log.Info("Listening on :8080")
	srv := api.New(repo, gcp, oauth2Config, teamProjectsMapping, accessMgr, authenticatorMiddleware, teamkatalogen.New(cfg.TeamkatalogenURL), prom(promErrs, repo.Metrics()), log)

	server := http.Server{
		Addr:    cfg.BindAddress,
		Handler: srv,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Warn("Shutdown error")
	}
}

func prom(cols ...prometheus.Collector) *prometheus.Registry {
	r := prometheus.NewRegistry()
	graphProm.RegisterOn(r)
	r.MustRegister(prometheus.NewGoCollector())
	r.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{ReportErrors: true}))
	r.MustRegister(cols...)

	return r
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

func runMetabase(ctx context.Context, log *logrus.Entry, cfg Config, repo *database.Repo, accessMgr graph.AccessManager) error {
	if cfg.MetabaseServiceAccountFile == "" {
		log.Info("metabase sync disabled")
		return nil
	}

	log.Info("metabase sync enabled")

	client := metabase.NewClient(cfg.MetabaseAPI, cfg.MetabaseUsername, cfg.MetabasePassword)

	sa, err := os.ReadFile(cfg.MetabaseServiceAccountFile)
	if err != nil {
		return err
	}

	metabaseSA := struct {
		ClientEmail string `json:"client_email"`
	}{}

	err = json.Unmarshal(sa, &metabaseSA)
	if err != nil {
		return err
	}

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return err
	}

	metabase := metabase.New(repo, client, accessMgr, string(sa), metabaseSA.ClientEmail, promErrs, iamService, log.WithField("subsystem", "metabase"))
	go metabase.Run(ctx, MetabaseUpdateFrequency)
	return nil
}

type noopDatasetEnricher struct{}

func (n noopDatasetEnricher) UpdateSchema(ctx context.Context, ds gensql.DatasourceBigquery) error {
	return nil
}
