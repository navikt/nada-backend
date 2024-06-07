package main

import (
	"context"
	"encoding/json"
	"net"
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
	"github.com/navikt/nada-backend/pkg/config/v2"
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

var configFilePath = flag.String("config", "config.yaml", "path to config file")

var promErrs = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "nada_backend",
	Name:      "errors",
}, []string{"location"})

const (
	TeamProjectsUpdateFrequency = 60 * time.Minute
	AccessEnsurerFrequency      = 5 * time.Minute
	MetabaseUpdateFrequency     = 1 * time.Hour
)

func main() {
	flag.Parse()

	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	fileParts, err := config.ProcessConfigPath(*configFilePath)
	if err != nil {
		log.WithError(err).Fatal("processing config path")
	}

	cfg, err := config.NewFileSystemLoader().Load(fileParts.FileName, fileParts.Path, "NADA", config.NewDefaultEnvBinder())
	if err != nil {
		log.WithError(err).Fatal("loading config")
	}

	err = cfg.Validate()
	if err != nil {
		log.WithError(err).Fatal("validating config")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	slackClient := slack.NewSlackClient(log, cfg.Slack.WebhookURL, cfg.Server.Hostname, cfg.Slack.Token)

	eventMgr := event.New(ctx)

	repo, err := database.New(
		cfg.Postgres.ConnectionString(),
		cfg.Postgres.Configuration.MaxIdleConnections,
		cfg.Postgres.Configuration.MaxOpenConnections,
		eventMgr,
		log.WithField("subsystem", "repo"),
		cfg.GCP.Project,
	)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	httpwithcache.SetDatabase(repo.GetDB())

	gcsClient, err := gcs.New(
		ctx,
		cfg.GCP.GCS.StoryBucketName,
		cfg.GCP.GCS.Endpoint,
		log.WithField("subsystem", "gcs"),
	)
	if err != nil {
		log.Fatal(err)
	}

	googleGroups, err := auth.NewGoogleGroups(
		ctx,
		cfg.GoogleGroups.CredentialsFile,
		cfg.GoogleGroups.ImpersonationSubject,
		log.WithField("subsystem", "googlegroups"),
	)
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
	var accessMgr access.AccessManager = access.Noop{}
	var pollyAPI polly.Polly = polly.NewMock(cfg.TreatmentCatalogue.APIURL)
	if !cfg.MockAuth {
		teamcatalogue = teamkatalogen.New(cfg.TeamsCatalogue.APIURL, repo.GetDB(), repo.Querier, log)

		teamProjectsUpdater = teamprojectsupdater.NewTeamProjectsUpdater(
			ctx,
			cfg.NaisConsole.APIURL,
			cfg.NaisConsole.APIKey,
			http.DefaultClient,
			repo,
		)
		go teamProjectsUpdater.Run(ctx, TeamProjectsUpdateFrequency)

		azureGroups := auth.NewAzureGroups(
			http.DefaultClient,
			cfg.Oauth.ClientID,
			cfg.Oauth.ClientSecret,
			cfg.Oauth.TenantID,
		)

		aauth := auth.NewAzure(
			cfg.Oauth.ClientID,
			cfg.Oauth.ClientSecret,
			cfg.Oauth.TenantID,
			cfg.Oauth.RedirectURL,
		)
		oauth2Config = aauth

		httpAPI = api.NewHTTP(oauth2Config, aauth.RedirectURL, cfg.LoginPage, cfg.Cookies, log.WithField("subsystem", "api"))
		authenticatorMiddleware = aauth.Middleware(aauth.KeyDiscoveryURL(), azureGroups, googleGroups, repo.GetDB())
		accessMgr = access.NewBigquery(cfg.GCP.BigQuery.Endpoint)
		pollyAPI = polly.New(cfg.TreatmentCatalogue.APIURL)
		amplitudeClient = amplitude.New(cfg.AmplitudeAPIKey, log.WithField("subsystem", "amplitude"))
	} else {
		teamProjectsUpdater, err = teamprojectsupdater.NewMockTeamProjectsUpdater(ctx, repo)
		if err != nil {
			log.Fatal(err)
		}
	}

	var bqClient bqclient.BQClient = bqclient.NewMock()
	if !cfg.SkipMetadataSync {
		datacatalogClient, err := bqclient.New(cfg.GCP.Project, cfg.GCP.BigQuery.PseudoViewsDatasetName, cfg.GCP.BigQuery.Endpoint)
		if err != nil {
			log.WithError(err).Fatal("creating datacatalog client")
		}

		bqClient = datacatalogClient

		bqClient, err = bqclient.New(cfg.GCP.Project, cfg.GCP.BigQuery.PseudoViewsDatasetName, cfg.GCP.BigQuery.Endpoint)
		if err != nil {
			log.WithError(err).Fatal("Creating bqclient")
		}
	}

	log.Infof("Listening on %s:%s", cfg.Server.Address, cfg.Server.Port)
	auth.Init(repo.GetDB())
	srv := api.New(
		httpAPI,
		authenticatorMiddleware,
		prom(repo.Metrics()...),
		cfg.API.AuthToken,
		cfg.GCP.Project,
		log,
	)
	service.Init(repo.GetDB(), teamcatalogue, log, teamProjectsUpdater.TeamProjectsMapping, eventMgr, slackClient, bqClient, pollyAPI, teamProjectsUpdater.TeamProjectsMapping, gcsClient, amplitudeClient)

	server := http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Address, cfg.Server.Port),
		Handler: srv,
	}

	err = createMetabaseSyncer(ctx, log.WithField("subsystem", "metabase"), repo, accessMgr, eventMgr, cfg)
	if err != nil {
		log.WithError(err).Fatal("running metabase")
	}

	go access_ensurer.NewEnsurer(nil, accessMgr, bqClient, googleGroups, cfg.GCP.Project, promErrs, log.WithField("subsystem", "accessensurer")).Run(ctx, AccessEnsurerFrequency)

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

func createMetabaseSyncer(ctx context.Context, log *logrus.Entry, repo *database.Repo, accessMgr access.AccessManager, eventMgr *event.Manager, cfg config.Config) error {
	if cfg.Metabase.CredentialsPath == "" {
		log.Info("metabase sync disabled")
		return nil
	}

	log.Info("metabase sync enabled")

	client := metabase.NewClient(
		cfg.Metabase.APIURL,
		cfg.Metabase.Username,
		cfg.Metabase.Password,
		cfg.Oauth.ClientID,
		cfg.Oauth.ClientSecret,
		cfg.Oauth.TenantID,
	)
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return err
	}

	sa, err := os.ReadFile(cfg.Metabase.CredentialsPath)
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

	metabase := metabase.New(repo, client, eventMgr, accessMgr, string(sa), metabaseSA.ClientEmail, promErrs, iamService, crmService, cfg.Metabase.GCPProject, log.WithField("subsystem", "metabase"))
	go metabase.Run(ctx, MetabaseUpdateFrequency)
	return nil
}
