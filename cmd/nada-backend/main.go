package main

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/nc"
	"github.com/navikt/nada-backend/pkg/service/core"
	apiclients "github.com/navikt/nada-backend/pkg/service/core/api"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/navikt/nada-backend/pkg/access_ensurer"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/metabase"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var configFilePath = flag.String("config", "config.yaml", "path to config file")

var promErrs = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "nada_backend",
	Name:      "errors",
}, []string{"location"})

const (
	TeamProjectsUpdateFrequency = 60 * time.Minute
	TeamProjectDelay            = 10 * time.Second
	AccessEnsurerFrequency      = 5 * time.Minute
	MetabaseUpdateFrequency     = 1 * time.Hour
)

func main() {
	flag.Parse()

	zlog := zerolog.New(os.Stdout).With().Timestamp().Logger()

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

	repo, err := database.New(
		cfg.Postgres.ConnectionString(),
		cfg.Postgres.Configuration.MaxIdleConnections,
		cfg.Postgres.Configuration.MaxOpenConnections,
		log.WithField("subsystem", "repo"),
	)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	tkFetcher := tk.New(cfg.TeamsCatalogue.APIURL, httpClient)
	ncFetcher := nc.New(cfg.NaisConsole.APIURL, cfg.NaisConsole.APIKey, cfg.NaisClusterName, httpClient)

	// FIXME: make this configurable
	cacher := cache.New(2*time.Hour, repo.GetDB(), zlog)

	// FIXME: make authentication configurable
	bqClient := bq.NewClient(cfg.BigQuery.Endpoint, true)

	stores := storage.NewStores(repo, cfg)
	apiClients := apiclients.NewClients(cacher, tkFetcher, ncFetcher, bqClient, cfg, log.WithField("subsystem", "api_clients"))
	services, err := core.NewServices(cfg, stores, apiClients)
	if err != nil {
		log.WithError(err).Fatal("setting up services")
	}

	teamProjectsUpdater := teamprojectsupdater.New(
		services.NaisConsoleService,
		zlog,
	)
	go teamProjectsUpdater.Run(ctx, TeamProjectDelay, TeamProjectsUpdateFrequency)

	googleGroups, err := auth.NewGoogleGroups(
		ctx,
		cfg.GoogleGroups.CredentialsFile,
		cfg.GoogleGroups.ImpersonationSubject,
		log.WithField("subsystem", "googlegroups"),
	)
	if err != nil {
		log.Fatal(err)
	}

	metabaseSynchronizer := metabase.New(services.MetaBaseService)
	go metabaseSynchronizer.Run(
		ctx,
		MetabaseUpdateFrequency,
		log.WithField("subsystem", "metabase"),
	)

	teamcatalogue := teamkatalogen.New(
		apiClients.TeamKatalogenAPI,
		stores.ProductAreaStorage,
		log,
	)
	go teamcatalogue.RunSyncer()

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

	httpAPI := api.NewHTTP(
		aauth,
		aauth.RedirectURL,
		cfg.LoginPage,
		cfg.Cookies,
		log.WithField("subsystem", "api"),
	)

	authenticatorMiddleware := aauth.Middleware(
		aauth.KeyDiscoveryURL(),
		azureGroups,
		googleGroups,
		repo.GetDB(),
	)

	h := handlers.NewHandlers(
		services,
		amplitude.New(cfg.AmplitudeAPIKey, log.WithField("subsystem", "amplitude")),
		cfg,
	)

	log.Infof("Listening on %s:%s", cfg.Server.Address, cfg.Server.Port)
	auth.Init(repo.GetDB())

	router := chi.NewRouter()

	routes.Add(router,
		routes.NewInsightProductRoutes(routes.NewInsightProductEndpoints(zlog, h.InsightProductHandler), authenticatorMiddleware),
		routes.NewAccessRoutes(routes.NewAccessEndpoints(zlog, h.AccessHandler), authenticatorMiddleware),
		routes.NewBigQueryRoutes(routes.NewBigQueryEndpoints(zlog, h.BigQueryHandler)),
		routes.NewDataProductsRoutes(routes.NewDataProductsEndpoints(zlog, h.DataProductsHandler), authenticatorMiddleware),
		routes.NewJoinableViewsRoutes(routes.NewJoinableViewsEndpoints(zlog, h.JoinableViewsHandler), authenticatorMiddleware),
		routes.NewKeywordRoutes(routes.NewKeywordEndpoints(zlog, h.KeywordsHandler), authenticatorMiddleware),
		routes.NewMetabaseRoutes(routes.NewMetabaseEndpoints(zlog, h.MetabaseHandler), authenticatorMiddleware),
		routes.NewPollyRoutes(routes.NewPollyEndpoints(zlog, h.PollyHandler)),
		routes.NewProductAreaRoutes(routes.NewProductAreaEndpoints(zlog, h.ProductAreasHandler), authenticatorMiddleware),
		routes.NewSearchRoutes(routes.NewSearchEndpoints(zlog, h.SearchHandler)),
		routes.NewSlackRoutes(routes.NewSlackEndpoints(zlog, h.SlackHandler)),
		routes.NewStoryRoutes(routes.NewStoryEndpoints(zlog, h.StoryHandler), authenticatorMiddleware),
		routes.NewTeamkatalogenRoutes(routes.NewTeamkatalogenEndpoints(zlog, h.TeamKatalogenHandler)),
		routes.NewTokensRoutes(routes.NewTokensEndpoints(zlog, h.TokenHandler), authenticatorMiddleware),
		routes.NewMetricsRoutes(routes.NewMetricsEndpoints(prom(repo.Metrics()...))),
		routes.NewUserRoutes(routes.NewUserEndpoints(zlog, h.UserHandler), authenticatorMiddleware),
		routes.NewAuthRoutes(routes.NewAuthEndpoints(httpAPI)),
	)

	server := http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Address, cfg.Server.Port),
		Handler: router,
	}

	go access_ensurer.NewEnsurer(
		googleGroups,
		cfg.BigQuery.CentralGCPProject,
		promErrs,
		stores.AccessStorage,
		services.MetaBaseService,
		stores.DataProductsStorage,
		stores.BigQueryStorage,
		apiClients.BigQueryAPI,
		services.BigQueryService,
		services.JoinableViewService,
		log.WithField("subsystem", "accessensurer"),
	).Run(ctx, AccessEnsurerFrequency)

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
