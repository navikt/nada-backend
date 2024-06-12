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
	"github.com/navikt/nada-backend/pkg/access_ensurer"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/config"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/httpwithcache"
	"github.com/navikt/nada-backend/pkg/metabase"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/slack"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

var promErrs = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "nada_backend",
	Name:      "errors",
}, []string{"location"})

const (
	TeamProjectsUpdateFrequency = 60 * time.Minute
	AccessEnsurerFrequency      = 5 * time.Minute
	MetabaseUpdateFrequency     = 1 * time.Hour
)

func init() {
	flag.StringVar(&config.Conf.BindAddress, "bind-address", config.Conf.BindAddress, "Bind address")
	flag.StringVar(&config.Conf.DBConnectionDSN, "db-connection-dsn", fmt.Sprintf("%v?sslmode=disable", getEnv("NAIS_DATABASE_NADA_BACKEND_NADA_URL", "postgres://postgres:postgres@127.0.0.1:5432/nada")), "database connection DSN")
	flag.StringVar(&config.Conf.Hostname, "hostname", os.Getenv("HOSTNAME"), "Hostname the application is served from")
	flag.StringVar(&config.Conf.OAuth2.ClientID, "oauth2-client-id", os.Getenv("AZURE_APP_CLIENT_ID"), "OAuth2 client ID")
	flag.StringVar(&config.Conf.OAuth2.ClientSecret, "oauth2-client-secret", os.Getenv("AZURE_APP_CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&config.Conf.OAuth2.TenantID, "oauth2-tenant-id", os.Getenv("AZURE_APP_TENANT_ID"), "OAuth2 azure tenant id")
	flag.StringVar(&config.Conf.TeamProjectsOutputURL, "team-projects-url", config.Conf.TeamProjectsOutputURL, "URL for json containing team projects")
	flag.StringVar(&config.Conf.TeamsToken, "teams-token", os.Getenv("GITHUB_READ_TOKEN"), "Token for accessing teams json")
	flag.StringVar(&config.Conf.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&config.Conf.CookieSecret, "cookie-secret", "", "Secret used when encrypting cookies")
	flag.StringVar(&config.Conf.TeamkatalogenURL, "teamkatalogen-url", config.Conf.TeamkatalogenURL, "Teamkatalog API URL")
	flag.BoolVar(&config.Conf.MockAuth, "mock-auth", false, "Use mock authentication")
	flag.StringVar(&config.Conf.GoogleAdminImpersonationSubject, "google-admin-subject", os.Getenv("GOOGLE_ADMIN_IMPERSONATION_SUBJECT"), "Subject to impersonate when accessing google admin apis")
	flag.StringVar(&config.Conf.ServiceAccountFile, "service-account-file", os.Getenv("GOOGLE_ADMIN_CREDENTIALS_PATH"), "Service account file for accessing google admin apis")
	flag.BoolVar(&config.Conf.SkipMetadataSync, "skip-metadata-sync", false, "Skip metadata sync")
	flag.StringVar(&config.Conf.MetabaseServiceAccountFile, "metabase-service-account-file", os.Getenv("METABASE_GOOGLE_CREDENTIALS_PATH"), "Service account file for metabase access to bigquery tables")
	flag.StringVar(&config.Conf.MetabaseUsername, "metabase-username", os.Getenv("METABASE_USERNAME"), "Username for metabase api")
	flag.StringVar(&config.Conf.MetabasePassword, "metabase-password", os.Getenv("METABASE_PASSWORD"), "Password for metabase api")
	flag.StringVar(&config.Conf.MetabaseAPI, "metabase-api", os.Getenv("METABASE_API"), "URL to Metabase API, including scheme and `/api`")
	flag.StringVar(&config.Conf.SlackUrl, "slack-url", os.Getenv("SLACK_URL"), "URL for slack webhook")
	flag.StringVar(&config.Conf.SlackToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token for slack app")
	flag.StringVar(&config.Conf.PollyURL, "polly-url", config.Conf.PollyURL, "URL for polly")
	flag.IntVar(&config.Conf.DBMaxIdleConn, "max-idle-conn", 3, "Maximum number of idle db connections")
	flag.IntVar(&config.Conf.DBMaxOpenConn, "max-open-conn", 5, "Maximum number of open db connections")
	flag.StringVar(&config.Conf.StoryBucketName, "story-bucket", os.Getenv("GCP_STORY_BUCKET_NAME"), "Name of the gcs bucket for story content")
	flag.StringVar(&config.Conf.ConsoleAPIKey, "console-api-key", os.Getenv("CONSOLE_API_KEY"), "API key for nais console")
	flag.StringVar(&config.Conf.AmplitudeAPIKey, "amplitude-api-key", os.Getenv("AMPLITUDE_API_KEY"), "API key for Amplitude")
	flag.StringVar(&config.Conf.CentralDataProject, "central-data-project", os.Getenv("CENTRAL_DATA_PROJECT"), "bigquery project for pseudo views")
	flag.StringVar(&config.Conf.PseudoDataset, "pseudo-dataset", "markedsplassen_pseudo", "bigquery dataset in producers' project for markedplassen saving pseudo views")
	flag.StringVar(&config.Conf.NadaTokenCreds, "nada-token-creds", os.Getenv("NADA_TOKEN_CREDS"), "Auth credentials for fetching nada tokens")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log := newLogger()
	slackClient := newSlackClient(log)

	eventMgr := event.New(ctx)

	repo, err := database.New(config.Conf.DBConnectionDSN, config.Conf.DBMaxIdleConn, config.Conf.DBMaxOpenConn, eventMgr, log.WithField("subsystem", "repo"), config.Conf.CentralDataProject)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	httpwithcache.SetDatabase(repo.GetDB())

	gcsClient, err := gcs.New(ctx, config.Conf.StoryBucketName, log.WithField("subsystem", "gcs"))
	if err != nil {
		log.Fatal(err)
	}

	googleGroups, err := auth.NewGoogleGroups(ctx, config.Conf.ServiceAccountFile, config.Conf.GoogleAdminImpersonationSubject, log.WithField("subsystem", "googlegroups"))
	if err != nil {
		log.Fatal(err)
	}

	mockHTTP := service.NewMockHTTP(repo, log.WithField("subsystem", "mockhttp"))
	var httpAPI api.HTTPAPI = mockHTTP

	authenticatorMiddleware := mockHTTP.Middleware

	var teamProjectsUpdater *teamprojectsupdater.TeamProjectsUpdater
	var oauth2Config api.OAuth2
	var teamcatalogue teamkatalogen.Teamkatalogen = teamkatalogen.NewMock()
	var amplitudeClient amplitude.Amplitude
	amplitudeClient = amplitude.NewMock()
	var accessMgr access.AccessManager = access.Noop{}
	var pollyAPI polly.Polly = polly.NewMock(config.Conf.PollyURL)
	if !config.Conf.MockAuth {
		teamcatalogue = teamkatalogen.New(config.Conf.TeamkatalogenURL, repo.GetDB(), repo.Querier, log)

		teamProjectsUpdater = teamprojectsupdater.NewTeamProjectsUpdater(ctx, config.Conf.ConsoleURL, config.Conf.ConsoleAPIKey, http.DefaultClient, repo)
		go teamProjectsUpdater.Run(ctx, TeamProjectsUpdateFrequency)

		azureGroups := auth.NewAzureGroups(http.DefaultClient, config.Conf.OAuth2.ClientID, config.Conf.OAuth2.ClientSecret, config.Conf.OAuth2.TenantID)

		aauth := auth.NewAzure(config.Conf.OAuth2.ClientID, config.Conf.OAuth2.ClientSecret, config.Conf.OAuth2.TenantID, config.Conf.Hostname)
		oauth2Config = aauth

		httpAPI = api.NewHTTP(oauth2Config, aauth.RedirectURL, log.WithField("subsystem", "api"))
		authenticatorMiddleware = aauth.Middleware(aauth.KeyDiscoveryURL(), azureGroups, googleGroups, repo.GetDB())
		accessMgr = access.NewBigquery()
		pollyAPI = polly.New(config.Conf.PollyURL)
		amplitudeClient = amplitude.New(config.Conf.AmplitudeAPIKey, log.WithField("subsystem", "amplitude"))
	} else {
		teamProjectsUpdater, err = teamprojectsupdater.NewMockTeamProjectsUpdater(ctx, repo)
		if err != nil {
			log.Fatal(err)
		}
	}
	var bqClient bqclient.BQClient = bqclient.NewMock()
	if !config.Conf.SkipMetadataSync {
		datacatalogClient, err := bqclient.New(ctx, config.Conf.CentralDataProject, config.Conf.PseudoDataset)
		if err != nil {
			log.WithError(err).Fatal("Creating datacatalog client")
		}

		bqClient = datacatalogClient

		bqClient, err = bqclient.New(ctx, config.Conf.CentralDataProject, config.Conf.PseudoDataset)
		if err != nil {
			log.WithError(err).Fatal("Creating bqclient")
		}
	}

	log.Info("Listening on :8080")
	auth.Init(repo.GetDB())
	srv := api.New(repo, gcsClient, teamcatalogue, httpAPI, authenticatorMiddleware, prom(repo.Metrics()...), amplitudeClient, config.Conf.NadaTokenCreds, log)
	service.Init(repo.GetDB(), teamcatalogue, log, teamProjectsUpdater.TeamProjectsMapping, eventMgr, slackClient, bqClient, pollyAPI, teamProjectsUpdater.TeamProjectsMapping, gcsClient, amplitudeClient)

	server := http.Server{
		Addr:    config.Conf.BindAddress,
		Handler: srv,
	}

	err = createMetabaseSyncer(ctx, log.WithField("subsystem", "metabase"), repo, accessMgr, eventMgr)
	if err != nil {
		log.WithError(err).Fatal("running metabase")
	}

	go access_ensurer.NewEnsurer(nil, accessMgr, bqClient, repo, googleGroups, config.Conf.CentralDataProject, promErrs, log.WithField("subsystem", "accessensurer")).Run(ctx, AccessEnsurerFrequency)

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

	l, err := logrus.ParseLevel(config.Conf.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
	return log
}

func newSlackClient(log *logrus.Logger) *slack.SlackClient {
	return slack.NewSlackClient(log, config.Conf.SlackUrl, config.Conf.Hostname, config.Conf.SlackToken)
}

func getEnv(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return fallback
}

func createMetabaseSyncer(ctx context.Context, log *logrus.Entry, repo *database.Repo, accessMgr access.AccessManager, eventMgr *event.Manager) error {
	if config.Conf.MetabaseServiceAccountFile == "" {
		log.Info("metabase sync disabled")
		return nil
	}

	log.Info("metabase sync enabled")

	client := metabase.NewClient(config.Conf.MetabaseAPI, config.Conf.MetabaseUsername, config.Conf.MetabasePassword, config.Conf.OAuth2.ClientID, config.Conf.OAuth2.ClientSecret, config.Conf.OAuth2.TenantID)
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return err
	}

	sa, err := os.ReadFile(config.Conf.MetabaseServiceAccountFile)
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

	metabase := metabase.New(repo, client, eventMgr, accessMgr, string(sa), metabaseSA.ClientEmail, promErrs, iamService, crmService, log.WithField("subsystem", "metabase"))
	go metabase.Run(ctx, MetabaseUpdateFrequency)
	return nil
}
