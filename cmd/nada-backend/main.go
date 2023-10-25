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
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/httpwithcache"
	"github.com/navikt/nada-backend/pkg/metabase"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/slack"
	"github.com/navikt/nada-backend/pkg/story"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/cloudresourcemanager/v1"
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
	TeamProjectsUpdateFrequency = 60 * time.Minute
	AccessEnsurerFrequency      = 5 * time.Minute
	MetabaseUpdateFrequency     = 1 * time.Hour
	StoryDraftCleanerFrequency  = 24 * time.Hour
)

func init() {
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.DBConnectionDSN, "db-connection-dsn", fmt.Sprintf("%v?sslmode=disable", getEnv("NAIS_DATABASE_NADA_BACKEND_NADA_URL", "postgres://postgres:postgres@127.0.0.1:5432/nada")), "database connection DSN")
	flag.StringVar(&cfg.Hostname, "hostname", os.Getenv("HOSTNAME"), "Hostname the application is served from")
	flag.StringVar(&cfg.OAuth2.ClientID, "oauth2-client-id", os.Getenv("AZURE_APP_CLIENT_ID"), "OAuth2 client ID")
	flag.StringVar(&cfg.OAuth2.ClientSecret, "oauth2-client-secret", os.Getenv("AZURE_APP_CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&cfg.OAuth2.TenantID, "oauth2-tenant-id", os.Getenv("AZURE_APP_TENANT_ID"), "OAuth2 azure tenant id")
	flag.StringVar(&cfg.TeamProjectsOutputURL, "team-projects-url", cfg.TeamProjectsOutputURL, "URL for json containing team projects")
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
	flag.StringVar(&cfg.SlackUrl, "slack-url", os.Getenv("SLACK_URL"), "URL for slack webhook")
	flag.StringVar(&cfg.SlackToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token for slack app")
	flag.StringVar(&cfg.PollyURL, "polly-url", cfg.PollyURL, "URL for polly")
	flag.IntVar(&cfg.DBMaxIdleConn, "max-idle-conn", 3, "Maximum number of idle db connections")
	flag.IntVar(&cfg.DBMaxOpenConn, "max-open-conn", 5, "Maximum number of open db connections")
	flag.StringVar(&cfg.QuartoStorageBucketName, "quarto-bucket", os.Getenv("GCP_QUARTO_STORAGE_BUCKET_NAME"), "Name of the gcs bucket for quarto stories")
	flag.StringVar(&cfg.ConsoleAPIKey, "console-api-key", os.Getenv("CONSOLE_API_KEY"), "API key for nais console")
	flag.StringVar(&cfg.AmplitudeAPIKey, "amplitude-api-key", os.Getenv("AMPLITUDE_API_KEY"), "API key for Amplitude")
	flag.StringVar(&cfg.CentralDataProject, "central-data-project", os.Getenv("CENTRAL_DATA_PROJECT"), "bigquery project for pseudo views")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log := newLogger()
	slackClient := newSlackClient(log)

	eventMgr := event.New(ctx)

	repo, err := database.New(cfg.DBConnectionDSN, cfg.DBMaxIdleConn, cfg.DBMaxOpenConn, eventMgr, log.WithField("subsystem", "repo"))
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	httpwithcache.SetDatabase(repo.GetDB())

	gcsClient, err := gcs.New(ctx, cfg.QuartoStorageBucketName, log.WithField("subsystem", "gcs"))
	if err != nil {
		log.Fatal(err)
	}

	mockHTTP := api.NewMockHTTP(repo, log.WithField("subsystem", "mockhttp"))
	var httpAPI api.HTTPAPI = mockHTTP
	authenticatorMiddleware := mockHTTP.Middleware

	var teamProjectsUpdater *teamprojectsupdater.TeamProjectsUpdater
	var oauth2Config api.OAuth2
	var accessMgr graph.AccessManager
	accessMgr = access.NewNoop()
	var teamcatalogue graph.Teamkatalogen = teamkatalogen.NewMock()
	var pollyAPI graph.Polly = polly.NewMock(cfg.PollyURL)
	var amplitudeClient amplitude.Amplitude
	amplitudeClient = amplitude.NewMock()
	if !cfg.MockAuth {
		teamcatalogue = teamkatalogen.New(cfg.TeamkatalogenURL)
		teamProjectsUpdater = teamprojectsupdater.NewTeamProjectsUpdater(ctx, cfg.ConsoleURL, cfg.ConsoleAPIKey, http.DefaultClient, repo)
		go teamProjectsUpdater.Run(ctx, TeamProjectsUpdateFrequency)

		azureGroups := auth.NewAzureGroups(http.DefaultClient, cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID)
		googleGroups, err := auth.NewGoogleGroups(ctx, cfg.ServiceAccountFile, cfg.GoogleAdminImpersonationSubject, log.WithField("subsystem", "googlegroups"))
		if err != nil {
			log.Fatal(err)
		}

		aauth := auth.NewAzure(cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID, cfg.Hostname)
		oauth2Config = aauth

		httpAPI = api.NewHTTP(oauth2Config, aauth.RedirectURL, repo, log.WithField("subsystem", "api"))
		authenticatorMiddleware = aauth.Middleware(aauth.KeyDiscoveryURL(), azureGroups, googleGroups, repo)
		accessMgr = access.NewBigquery()
		pollyAPI = polly.New(cfg.PollyURL)
		amplitudeClient = amplitude.New(cfg.AmplitudeAPIKey, log.WithField("subsystem", "amplitude"))
	} else {
		teamProjectsUpdater, err = teamprojectsupdater.NewMockTeamProjectsUpdater(ctx, repo)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := runMetabase(ctx, log.WithField("subsystem", "metabase"), cfg, repo, accessMgr, eventMgr); err != nil {
		log.WithError(err).Fatal("running metabase")
	}

	go access.NewEnsurer(repo, accessMgr, promErrs, log.WithField("subsystem", "accessensurer")).Run(ctx, AccessEnsurerFrequency)

	var gcp graph.Bigquery = bigquery.NewMock()
	if !cfg.SkipMetadataSync {
		datacatalogClient, err := bigquery.New(ctx, cfg.CentralDataProject)
		if err != nil {
			log.WithError(err).Fatal("Creating datacatalog client")
		}

		gcp = datacatalogClient
	}

	go story.NewDraftCleaner(repo, log.WithField("subsystem", "storydraftcleaner")).Run(ctx, StoryDraftCleanerFrequency)

	log.Info("Listening on :8080")
	gqlServer := graph.New(repo, gcp, teamProjectsUpdater.TeamProjectsMapping, accessMgr, teamcatalogue, slackClient, pollyAPI, log.WithField("subsystem", "graph"))
	srv := api.New(repo, gcsClient, httpAPI, authenticatorMiddleware, gqlServer, prom(repo.Metrics()...), amplitudeClient, log)

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
	r.MustRegister(promErrs)
	r.MustRegister(prometheus.NewGoCollector())
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

func newSlackClient(log *logrus.Logger) *slack.SlackClient {
	return slack.NewSlackClient(log, cfg.SlackUrl, cfg.Hostname, cfg.SlackToken)
}

func getEnv(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return fallback
}

func runMetabase(ctx context.Context, log *logrus.Entry, cfg Config, repo *database.Repo, accessMgr graph.AccessManager, eventMgr *event.Manager) error {
	if cfg.MetabaseServiceAccountFile == "" {
		log.Info("metabase sync disabled")
		return nil
	}

	log.Info("metabase sync enabled")

	client := metabase.NewClient(cfg.MetabaseAPI, cfg.MetabaseUsername, cfg.MetabasePassword, cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret, cfg.OAuth2.TenantID)
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return err
	}

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

	metabase := metabase.New(repo, client, accessMgr, eventMgr, string(sa), metabaseSA.ClientEmail, promErrs, iamService, crmService, log.WithField("subsystem", "metabase"))
	go metabase.Run(ctx, MetabaseUpdateFrequency)
	return nil
}
