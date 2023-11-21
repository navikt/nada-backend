//go:build integration_test

package e2etests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/slack"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
	resource.Expire(120) // setting resource timeout as postgres container is not terminated automatically

	port, err := findAvailableHostPort()
	if err != nil {
		log.Fatalf("Could not start gcs resource: %s", err)
	}

	gcsResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "fsouza/fake-gcs-server",
		Tag:        "1.44",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"4443/tcp": {{HostIP: "localhost", HostPort: strconv.Itoa(port) + "/tcp"}},
		},
		Entrypoint: []string{"/bin/fake-gcs-server", "-scheme", "http", "-public-host", "localhost:" + strconv.Itoa(port)},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("could not start gcs resource %v", err)
	}

	os.Setenv("STORAGE_EMULATOR_HOST", fmt.Sprintf("http://localhost:%v/storage/v1/", gcsResource.GetPort("4443/tcp")))
	defer os.Unsetenv("STORAGE_EMULATOR_HOST")

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

	gcsClient, err := gcs.New(context.Background(), quartoBucket, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		panic(err)
	}

	repo, err = database.New(dbString, 2, 0, &event.Manager{}, logrus.NewEntry(logrus.StandardLogger()), "nav-central-data-dev-e170")
	if err != nil {
		panic(err)
	}

	tpu, err := teamprojectsupdater.NewMockTeamProjectsUpdater(context.Background(), repo)
	if err != nil {
		panic(err)
	}

	promReg := prometheus.NewRegistry()
	graphProm.RegisterOn(promReg)

	gqlServer := graph.New(
		repo,
		bigquery.NewMock(),
		tpu.TeamProjectsMapping,
		access.NewNoop(),
		teamkatalogen.NewMock(),
		slack.NewMockSlackClient(logrus.StandardLogger()),
		polly.NewMock("https://some.url"),
		"",
		logrus.StandardLogger().WithField("subsystem", "graphql"),
	)
	srv := api.New(
		repo,
		gcsClient,
		teamkatalogen.NewMock(),
		&mockAuthHandler{},
		auth.MockJWTValidatorMiddleware(),
		gqlServer,
		prometheus.NewRegistry(),
		amplitude.NewMock(),
		logrus.StandardLogger(),
	)

	server = httptest.NewServer(srv)

	ctx := context.Background()
	storyID, err := createStory(ctx, repo)
	if err != nil {
		log.Fatalf("Could not create story draft for e2e tests: %s", err)
	}
	code := m.Run()

	if err := deleteStory(ctx, repo, storyID); err != nil {
		log.Fatalf("Could not delete story draft after e2e tests: %s", err)
	}

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	if err := pool.Purge(gcsResource); err != nil {
		log.Fatalf("Could not purge gcs resource: %s", err)
	}

	os.Exit(code)
}

func createStory(ctx context.Context, repo *database.Repo) (uuid.UUID, error) {
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

	story, err := repo.PublishStory(ctx, models.NewStory{
		ID:            draftID,
		Group:         "team@nav.no",
		Name:          "mystory",
		Keywords:      []string{},
		ProductAreaID: stringToPtr("Mocked-001"),
		TeamID:        stringToPtr("team"),
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	return story.ID, nil
}

func deleteStory(ctx context.Context, repo *database.Repo, storyID uuid.UUID) error {
	return repo.DeleteStory(ctx, storyID)
}

func findAvailableHostPort() (int, error) {
	if a, err := net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return 0, fmt.Errorf("could not find a port")
}

func stringToPtr(s string) *string {
	return &s
}

type noopDatasetEnricher struct{}

func (n noopDatasetEnricher) UpdateSchema(ctx context.Context, ds gensql.DatasourceBigquery) error {
	return nil
}

type mockAuthHandler struct{}

func (m *mockAuthHandler) Login(w http.ResponseWriter, r *http.Request)    {}
func (m *mockAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {}
func (m *mockAuthHandler) Logout(w http.ResponseWriter, r *http.Request)   {}
