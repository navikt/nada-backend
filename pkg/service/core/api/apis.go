package api

import (
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	slackapi "github.com/navikt/nada-backend/pkg/service/core/api/slack"
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
}

func NewClients(
	cfg config.Config,
	log *logrus.Entry,
) *Clients {
	return &Clients{
		BigQueryAPI: gcp.NewBigQueryAPI(
			cfg.GCP.Project,
			cfg.GCP.Region,
			cfg.GCP.BigQuery.Endpoint,
			cfg.GCP.BigQuery.PseudoViewsDatasetName,
		),
		StoryAPI: gcp.NewStoryAPI(
			cfg.GCP.Project,
			cfg.GCP.GCS.StoryBucketName,
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
		TeamKatalogenAPI: httpapi.NewTeamKatalogenAPI(
			cfg.TeamsCatalogue.APIURL,
		),
		SlackAPI: slackapi.NewSlackAPI(
			cfg.Slack.WebhookURL,
			cfg.Server.Hostname,
			cfg.Slack.Token,
		),
	}
}