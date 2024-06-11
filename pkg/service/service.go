package service

import (
	"database/sql"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/slack"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/sirupsen/logrus"
)

var sqldb *sql.DB
var queries *gensql.Queries
var tkClient teamkatalogen.Teamkatalogen
var log *logrus.Logger
var teamProjectsMapping *auth.TeamProjectsMapping
var accessManager access.Bigquery
var slackClient *slack.SlackClient
var pollyClient polly.Polly
var bq bqclient.BQClient
var gcpProjects *auth.TeamProjectsMapping
var amplitudeClient amplitude.Amplitude
var gcsClient *gcs.Client

func Init(db *sql.DB, tk teamkatalogen.Teamkatalogen, l *logrus.Logger, projects *auth.TeamProjectsMapping,
	sc *slack.SlackClient, b bqclient.BQClient, polly polly.Polly, gcpproj *auth.TeamProjectsMapping, gcs *gcs.Client, am amplitude.Amplitude) {
	tkClient = tk
	log = l
	teamProjectsMapping = projects
	sqldb = db
	queries = gensql.New(sqldb)
	bq = b
	slackClient = sc
	pollyClient = polly
	gcpProjects = gcpproj
	amplitudeClient = am
	gcsClient = gcs
}
