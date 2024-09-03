package api

import (
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/cs"
	"github.com/navikt/nada-backend/pkg/nc"
	"github.com/navikt/nada-backend/pkg/sa"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	slackapi "github.com/navikt/nada-backend/pkg/service/core/api/slack"
	"github.com/navikt/nada-backend/pkg/service/core/cache/postgres"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
)

type Clients struct {
	BigQueryAPI       service.BigQueryAPI
	StoryAPI          service.StoryAPI
	ServiceAccountAPI service.ServiceAccountAPI
	MetaBaseAPI       service.MetabaseAPI
	PollyAPI          service.PollyAPI
	TeamKatalogenAPI  service.TeamKatalogenAPI
	SlackAPI          service.SlackAPI
	NaisConsoleAPI    service.NaisConsoleAPI
}

func NewClients(
	cache cache.Cacher,
	tkFetcher tk.Fetcher,
	ncFetcher nc.Fetcher,
	bqClient bq.Operations,
	csClient cs.Operations,
	saClient sa.Operations,
	cfg config.Config,
	log zerolog.Logger,
) *Clients {
	tkAPI := httpapi.NewTeamKatalogenAPI(tkFetcher, log)
	tkAPICacher := postgres.NewTeamKatalogenCache(tkAPI, cache)

	return &Clients{
		BigQueryAPI: gcp.NewBigQueryAPI(
			cfg.BigQuery.CentralGCPProject,
			cfg.BigQuery.GCPRegion,
			cfg.BigQuery.TeamProjectPseudoViewsDatasetName,
			bqClient,
		),
		StoryAPI: gcp.NewStoryAPI(
			csClient,
			log.With().Str("component", "story").Logger(),
		),
		ServiceAccountAPI: gcp.NewServiceAccountAPI(saClient),
		MetaBaseAPI: httpapi.NewMetabaseHTTP(
			cfg.Metabase.APIURL,
			cfg.Metabase.Username,
			cfg.Metabase.Password,
			cfg.Metabase.BigQueryDatabase.APIEndpointOverride,
			cfg.Metabase.BigQueryDatabase.DisableAuth,
			cfg.Debug,
			log.With().Str("component", "metabase").Logger(),
		),
		PollyAPI: httpapi.NewPollyAPI(
			cfg.TreatmentCatalogue.PurposeURL,
			cfg.TreatmentCatalogue.APIURL,
		),
		TeamKatalogenAPI: tkAPICacher,
		SlackAPI: slackapi.NewSlackAPI(
			cfg.Slack.WebhookURL,
			cfg.Slack.Token,
		),
		NaisConsoleAPI: httpapi.NewNaisConsoleAPI(
			ncFetcher,
		),
	}
}
