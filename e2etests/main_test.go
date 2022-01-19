//go:build integration_test

package e2etests

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	graphProm "github.com/99designs/gqlgen-contrib/prometheus"
	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/ory/dockertest/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	repo   *database.Repo
	server *httptest.Server
)

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
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

	repo, err = database.New(dbString, &event.Manager{}, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		panic(err)
	}

	promReg := prometheus.NewRegistry()
	graphProm.RegisterOn(promReg)

	gqlServer := graph.New(
		repo,
		bigquery.NewMock(),
		&auth.MockTeamProjectsUpdater,
		access.NewNoop(),
		teamkatalogen.NewMock(),
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
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

type noopDatasetEnricher struct{}

func (n noopDatasetEnricher) UpdateSchema(ctx context.Context, ds gensql.DatasourceBigquery) error {
	return nil
}

type mockAuthHandler struct{}

func (m *mockAuthHandler) Login(w http.ResponseWriter, r *http.Request)    {}
func (m *mockAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {}
func (m *mockAuthHandler) Logout(w http.ResponseWriter, r *http.Request)   {}
