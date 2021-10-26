//go:build integration_test

package e2etests

import (
	"context"
	"database/sql"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/navikt/nada-backend/pkg/access"
	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/ory/dockertest/v3"
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

	repo, err = database.New(dbString, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		panic(err)
	}
	srv := api.New(repo, &mockGCP{}, nil, &auth.MockTeamProjectsUpdater, access.NewMock(), auth.MockJWTValidatorMiddleware(), logrus.NewEntry(logrus.StandardLogger()))

	server = httptest.NewServer(srv)
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

type mockGCP struct{}

func (m *mockGCP) GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error) {
	return nil, nil
}

func (m *mockGCP) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
	return nil, nil
}
