//go:build integration_test

package api

import (
	"context"
	"database/sql"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	server *httptest.Server
	client *openapi.Client
	repo   *database.Repo
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

	startServer(dbString)
	defer server.Close()

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

type mockOauth struct{}

func (m *mockOauth) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return nil, nil
}
func (m *mockOauth) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string { return "" }
func (m *mockOauth) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return nil, nil
}

func startServer(connString string) {
	var err error
	repo, err = database.New(connString)
	if err != nil {
		log.Fatal(err)
	}

	router := NewRouter(repo, &mockOauth{}, logrus.StandardLogger().WithField("", ""), &auth.MockTeamProjectsUpdater, nil, auth.MockJWTValidatorMiddleware())
	server = httptest.NewServer(router)

	client, err = openapi.NewClient(server.URL + "/api")
	if err != nil {
		log.Fatal(err)
	}
}
