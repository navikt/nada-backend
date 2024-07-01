package api

import (
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/nc"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	slackapi "github.com/navikt/nada-backend/pkg/service/core/api/slack"
	"github.com/navikt/nada-backend/pkg/service/core/cache/postgres"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/sirupsen/logrus"
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
	cfg config.Config,
	log *logrus.Entry,
) *Clients {
	tkAPI := httpapi.NewTeamKatalogenAPI(tkFetcher)
	tkAPICacher := postgres.NewTeamKatalogenCache(tkAPI, cache)

	return &Clients{
		BigQueryAPI: gcp.NewBigQueryAPI(
			cfg.BigQuery.CentralGCPProject,
			cfg.BigQuery.GCPRegion,
			cfg.BigQuery.TeamProjectPseudoViewsDatasetName,
			bqClient,
		),
		StoryAPI: gcp.NewStoryAPI(
			cfg.GCS.CentralGCPProject,
			cfg.GCS.StoryBucketName,
			log,
		),
		ServiceAccountAPI: gcp.NewServiceAccountAPI(),
		MetaBaseAPI: httpapi.NewMetabaseHTTP(
			cfg.Metabase.APIURL,
			cfg.Metabase.Username,
			cfg.Metabase.Password,
			cfg.Oauth.ClientID,
			cfg.Oauth.ClientSecret,
			cfg.Oauth.TenantID,
		),
		PollyAPI: httpapi.NewPollyAPI(
			cfg.TreatmentCatalogue.APIURL,
			cfg.TreatmentCatalogue.PurposeURL,
		),
		TeamKatalogenAPI: tkAPICacher,
		SlackAPI: slackapi.NewSlackAPI(
			cfg.Slack.WebhookURL,
			cfg.Server.Hostname,
			cfg.Slack.Token,
		),
		NaisConsoleAPI: httpapi.NewNaisConsoleAPI(
			ncFetcher,
		),
	}
}
