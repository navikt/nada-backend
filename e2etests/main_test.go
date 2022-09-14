//go:build integration_test

package e2etests

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/slack"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/ory/dockertest/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	repo   *database.Repo
	server *httptest.Server
)

func TestMain(m *testing.M) {
	dockerHost := os.Getenv("HOME") + "/.colima/docker.sock"
	_, err := os.Stat(dockerHost)
	if err != nil {
		// uses a sensible default on windows (tcp/http) and linux/osx (socket)
		dockerHost = ""
	} else {
		dockerHost = "unix://" + dockerHost
	}

	pool, err := dockertest.NewPool(dockerHost)
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("postgres", "12", []string{"POSTGRES_PASSWORD=postgres", "POSTGRES_DB=nada"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	var dbString string
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		dbString = "user=postgres dbname=nada sslmode=disable password=postgres host=localhost port=" + resource.GetPort("5432/tcp")
		db, err := sql.Open("postgres", dbString)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	repo, err = database.New(dbString, 2, 0, &event.Manager{}, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		panic(err)
	}

	promReg := prometheus.NewRegistry()
	graphProm.RegisterOn(promReg)

	gqlServer := graph.New(
		repo,
		bigquery.NewMock(),
		teamprojectsupdater.NewMockTeamProjectsUpdater().TeamProjectsMapping,
		access.NewNoop(),
		teamkatalogen.NewMock(),
		slack.NewMockSlackClient(logrus.StandardLogger()),
		polly.NewMock("https://some.url"),
		logrus.StandardLogger().WithField("subsystem", "graphql"),
	)
	srv := api.New(
		repo,
		&mockAuthHandler{},
		auth.MockJWTValidatorMiddleware(),
		gqlServer,
		prometheus.NewRegistry(), logrus.StandardLogger(),
	)

	server = httptest.NewServer(srv)

	ctx := context.Background()
	draftID, err := createStoryDraft(ctx, repo)
	if err != nil {
		log.Fatalf("Could not create story draft for e2e tests: %s", err)
	}
	code := m.Run()

	if err := deleteStoryDraft(ctx, repo, draftID); err != nil {
		log.Fatalf("Could not delete story draft after e2e tests: %s", err)
	}

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createStoryDraft(ctx context.Context, repo *database.Repo) (uuid.UUID, error) {
	headerView := map[string]interface{}{
		"content": "Header",
		"level":   1,
	}
	headerBytes, err := json.Marshal(headerView)
	if err != nil {
		return uuid.UUID{}, err
	}

	mdView := map[string]interface{}{
		"content": "Markdown description",
	}
	mdBytes, err := json.Marshal(mdView)
	if err != nil {
		return uuid.UUID{}, err
	}

	draftID, err := repo.CreateStoryDraft(ctx, &models.DBStory{
		Name: "mystory",
		Views: []models.DBStoryView{
			{Type: "header", Spec: headerBytes},
			{Type: "markdown", Spec: mdBytes},
		},
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return draftID, nil
}

func deleteStoryDraft(ctx context.Context, repo *database.Repo, draftID uuid.UUID) error {
	return repo.DeleteStoryDraft(ctx, draftID)
}

type noopDatasetEnricher struct{}

func (n noopDatasetEnricher) UpdateSchema(ctx context.Context, ds gensql.DatasourceBigquery) error {
	return nil
}

type mockAuthHandler struct{}

func (m *mockAuthHandler) Login(w http.ResponseWriter, r *http.Request)    {}
func (m *mockAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {}
func (m *mockAuthHandler) Logout(w http.ResponseWriter, r *http.Request)   {}
