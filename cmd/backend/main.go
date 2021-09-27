package main

import (
	"context"
	"fmt"
	"github.com/navikt/datakatalogen/backend/accessensurer"
	"github.com/navikt/datakatalogen/backend/auth"
	firestore "github.com/navikt/datakatalogen/backend/firestore"
	"github.com/navikt/datakatalogen/backend/iam"
	"github.com/navikt/datakatalogen/backend/teamprojectsupdater"
	"net/http"
	"os"
	"time"

	"github.com/navikt/datakatalogen/backend/logger"

	"github.com/navikt/datakatalogen/backend/api"
	"github.com/navikt/datakatalogen/backend/config"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var cfg = config.DefaultConfig()

const (
	TeamsUpdateFrequency        = 5 * time.Minute
	EnsureAccessUpdateFrequency = 5 * time.Minute
	TeamProjectsUpdateFrequency = 5 * time.Minute
)

func init() {
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.OAuth2.ClientID, "oauth2-client-id", os.Getenv("AZURE_APP_CLIENT_ID"), "OAuth2 client ID")
	flag.StringVar(&cfg.OAuth2.ClientSecret, "oauth2-client-secret", os.Getenv("AZURE_APP_CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&cfg.OAuth2.TenantID, "oauth2-tenant-id", os.Getenv("AZURE_APP_TENANT_ID"), "Azure tenant id")
	flag.StringVar(&cfg.Firestore.GoogleProjectID, "firestore-google-project-id", os.Getenv("FIRESTORE_GOOGLE_PROJECT_ID"), "Firestore Google project ID")
	flag.StringVar(&cfg.Firestore.DataproductsCollection, "dataproducts-collection", os.Getenv("DATAPRODUCTS_COLLECTION"), "Dataproducts collection name")
	flag.StringVar(&cfg.Firestore.AccessUpdatesCollection, "access-updates-collection", os.Getenv("ACCESS_UPDATES_COLLECTION"), "Access updates collection name")
	flag.StringVar(&cfg.Hostname, "hostname", os.Getenv("HOSTNAME"), "Hostname the application is served from")
	flag.StringVar(&cfg.TeamsURL, "teams-url", cfg.TeamsURL, "URL for json containing teams and UUIDs")
	flag.StringVar(&cfg.ProdTeamProjectsOutputURL, "prod-team-projects-url", cfg.ProdTeamProjectsOutputURL, "URL for json containing prod team projects")
	flag.StringVar(&cfg.DevTeamProjectsOutputURL, "dev-team-projects-url", cfg.DevTeamProjectsOutputURL, "URL for json containing dev team projects")
	flag.StringVar(&cfg.TeamsToken, "teams-token", os.Getenv("GITHUB_READ_TOKEN"), "Token for accessing teams json")
	flag.StringVar(&cfg.State, "state", os.Getenv("DP_STATE"), "State to ensure consistency between OAuth2 requests")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "Run in development mode")
	flag.Parse()

	logger.Setup(cfg.LogLevel)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firestore, err := firestore.New(ctx, cfg.Firestore.GoogleProjectID, cfg.Firestore.DataproductsCollection, cfg.Firestore.AccessUpdatesCollection)
	if err != nil {
		log.Fatalf("Initializing firestore client: %v", err)
	}

	teamUUIDs := make(map[string]string)
	go auth.UpdateTeams(ctx, teamUUIDs, cfg.TeamsURL, cfg.TeamsToken, TeamsUpdateFrequency)

	teamProjectsMapping := make(map[string][]string)
	go teamprojectsupdater.New(ctx, teamProjectsMapping, cfg, TeamProjectsUpdateFrequency, nil).Run()

	iam := iam.New(ctx)
	go accessensurer.New(ctx, cfg, firestore, iam, EnsureAccessUpdateFrequency).Run()

	api := api.New(firestore, iam, cfg, teamUUIDs, teamProjectsMapping)
	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, api))
}
