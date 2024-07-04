package main

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/cs"
	"github.com/navikt/nada-backend/pkg/nc"
	"github.com/navikt/nada-backend/pkg/service/core"
	apiclients "github.com/navikt/nada-backend/pkg/service/core/api"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/navikt/nada-backend/pkg/syncers/access_ensurer"
	"github.com/navikt/nada-backend/pkg/syncers/metabase"
	"github.com/navikt/nada-backend/pkg/syncers/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/syncers/teamprojectsupdater"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/prometheus/client_golang/prometheus"
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
	TeamKatalogenFrequency      = 1 * time.Hour
)

func main() {
	flag.Parse()

	zlog := zerolog.New(os.Stdout).With().Timestamp().Logger()

	fileParts, err := config.ProcessConfigPath(*configFilePath)
	if err != nil {
		zlog.Fatal().Err(err).Msg("processing config path")
	}

	cfg, err := config.NewFileSystemLoader().Load(fileParts.FileName, fileParts.Path, "NADA", config.NewDefaultEnvBinder())
	if err != nil {
		zlog.Fatal().Err(err).Msg("loading config")
	}

	err = cfg.Validate()
	if err != nil {
		zlog.Fatal().Err(err).Msg("validating config")
	}

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	zlog = zlog.Level(level)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	repo, err := database.New(
		cfg.Postgres.ConnectionString(),
		cfg.Postgres.Configuration.MaxIdleConnections,
		cfg.Postgres.Configuration.MaxOpenConnections,
	)
	if err != nil {
		zlog.Fatal().Err(err).Msg("setting up database")
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	tkFetcher := tk.New(cfg.TeamsCatalogue.APIURL, httpClient)
	ncFetcher := nc.New(cfg.NaisConsole.APIURL, cfg.NaisConsole.APIKey, cfg.NaisClusterName, httpClient)

	// FIXME: make this configurable
	cacher := cache.New(2*time.Hour, repo.GetDB(), zlog.With().Str("subsystem", "cache").Logger())

	// FIXME: make authentication configurable
	bqClient := bq.NewClient(cfg.BigQuery.Endpoint, true)

	csClient, err := cs.New(ctx, cfg.GCS.StoryBucketName)
	if err != nil {
		zlog.Fatal().Err(err).Msg("setting up cloud storage")
	}

	stores := storage.NewStores(repo, cfg, zlog.With().Str("subsystem", "stores").Logger())
	apiClients := apiclients.NewClients(
		cacher,
		tkFetcher,
		ncFetcher,
		bqClient,
		csClient,
		cfg,
		zlog.With().Str("subsystem", "api_clients").Logger(),
	)
	services, err := core.NewServices(cfg, stores, apiClients, zlog.With().Str("subsystem", "services").Logger())
	if err != nil {
		zlog.Fatal().Err(err).Msg("setting up services")
	}

	teamProjectsUpdater := teamprojectsupdater.New(
		services.NaisConsoleService,
		zlog.With().Str("subsystem", "teamprojectsupdater").Logger(),
	)
	go teamProjectsUpdater.Run(ctx, TeamProjectDelay, TeamProjectsUpdateFrequency)

	googleGroups, err := auth.NewGoogleGroups(
		ctx,
		cfg.GoogleGroups.CredentialsFile,
		cfg.GoogleGroups.ImpersonationSubject,
	)
	if err != nil {
		zlog.Fatal().Err(err).Msg("setting up google groups")
	}

	metabaseSynchronizer := metabase.New(services.MetaBaseService)
	go metabaseSynchronizer.Run(
		ctx,
		MetabaseUpdateFrequency,
		zlog.With().Str("subsystem", "metabase_sync").Logger(),
	)

	teamcatalogue := teamkatalogen.New(
		apiClients.TeamKatalogenAPI,
		stores.ProductAreaStorage,
		zlog.With().Str("subsystem", "teamkatalogen_sync").Logger(),
	)
	go teamcatalogue.Run(ctx, TeamKatalogenFrequency)

	azureGroups := auth.NewAzureGroups(
		http.DefaultClient,
		cfg.Oauth.ClientID,
		cfg.Oauth.ClientSecret,
		cfg.Oauth.TenantID,
		zlog.With().Str("subsystem", "azure_groups").Logger(),
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
		zlog.With().Str("subsystem", "api").Logger(),
	)

	authenticatorMiddleware := aauth.Middleware(
		aauth.KeyDiscoveryURL(),
		azureGroups,
		googleGroups,
		repo.GetDB(),
		zlog.With().Str("subsystem", "auth").Logger(),
	)

	// FIXME: hook up amplitude again, but as a middleware
	h := handlers.NewHandlers(
		services,
		cfg,
		zlog.With().Str("subsystem", "handlers").Logger(),
	)

	zlog.Info().Msgf("listening on %s:%s", cfg.Server.Address, cfg.Server.Port)
	auth.Init(repo.GetDB())

	router := chi.NewRouter()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		zlog.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("not found")
		w.WriteHeader(http.StatusNotFound)
	})

	routes.Add(router,
		routes.NewInsightProductRoutes(routes.NewInsightProductEndpoints(zlog, h.InsightProductHandler), authenticatorMiddleware),
		routes.NewAccessRoutes(routes.NewAccessEndpoints(zlog, h.AccessHandler), authenticatorMiddleware),
		routes.NewBigQueryRoutes(routes.NewBigQueryEndpoints(zlog, h.BigQueryHandler)),
		routes.NewDataProductsRoutes(routes.NewDataProductsEndpoints(zlog, h.DataProductsHandler), authenticatorMiddleware),
		routes.NewJoinableViewsRoutes(routes.NewJoinableViewsEndpoints(zlog, h.JoinableViewsHandler), authenticatorMiddleware),
		routes.NewKeywordRoutes(routes.NewKeywordEndpoints(zlog, h.KeywordsHandler), authenticatorMiddleware),
		routes.NewMetabaseRoutes(routes.NewMetabaseEndpoints(zlog, h.MetabaseHandler), authenticatorMiddleware),
		routes.NewPollyRoutes(routes.NewPollyEndpoints(zlog, h.PollyHandler)),
		routes.NewProductAreaRoutes(routes.NewProductAreaEndpoints(zlog, h.ProductAreasHandler)),
		routes.NewSearchRoutes(routes.NewSearchEndpoints(zlog, h.SearchHandler)),
		routes.NewSlackRoutes(routes.NewSlackEndpoints(zlog, h.SlackHandler)),
		routes.NewStoryRoutes(routes.NewStoryEndpoints(zlog, h.StoryHandler), authenticatorMiddleware, h.StoryHandler.NadaTokenMiddleware),
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
		zlog.With().Str("subsystem", "accessensurer").Logger(),
	).Run(ctx, AccessEnsurerFrequency)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			zlog.Fatal().Err(err).Msg("ListenAndServe")
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		zlog.Warn().Err(err).Msg("Shutdown error")
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
