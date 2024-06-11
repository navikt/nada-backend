//go:build integration_test

package teamprojectsupdater

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/navikt/nada-backend/pkg/metabase"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
)

func TestTeamProjectsUpdater(t *testing.T) {
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

	repo, err := database.New(dbString, 2, 0, &metabase.Manager{}, logrus.NewEntry(logrus.StandardLogger()), "nav-central-data-dev-e170")
	if err != nil {
		panic(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		teamsData := `
{
    "data": {
        "teams": {
            "nodes": [
                {
                    "slug": "team-a",
                    "environments": [
                        {
                            "gcpProjectID": "a-dev",
                            "name": "dev-gcp"
                        }
                    ]
                },
                {
                    "slug": "team-b",
                    "environments": [
                        {
                            "gcpProjectID": "b-dev",
                            "name": "dev-gcp"
                        }
                    ]
                },
                {
                    "slug": "team-c",
                    "environments": [
                        {
                            "gcpProjectID": "c-dev",
                            "name": "dev-gcp"
                        }
                    ]
                }
            ],
            "pageInfo": {
                "hasNextPage": false
            }
        }
    }
}`

		fmt.Fprintln(writer, string(teamsData))
	}))

	tup := NewTeamProjectsUpdater(context.TODO(), server.URL, "apiKey", server.Client(), repo)
	if err := tup.FetchTeamGoogleProjectsMapping(context.TODO()); err != nil {
		panic(err)
	}

	if len(tup.TeamProjectsMapping.TeamProjects) != 3 {
		t.Errorf("got: %v, want: %v", len(tup.TeamProjectsMapping.TeamProjects), 3)
	}
	if tup.TeamProjectsMapping.TeamProjects["team-a@nav.no"] != "a-dev" {
		t.Errorf("got: %v, want: %v", tup.TeamProjectsMapping.TeamProjects["team-a@nav.no"], "a-dev")
	}
	if tup.TeamProjectsMapping.TeamProjects["team-b@nav.no"] != "b-dev" {
		t.Errorf("got: %v, want: %v", tup.TeamProjectsMapping.TeamProjects["team-b@nav.no"], "b-dev")
	}
	if tup.TeamProjectsMapping.TeamProjects["team-c@nav.no"] != "c-dev" {
		t.Errorf("got: %v, want: %v", tup.TeamProjectsMapping.TeamProjects["team-c@nav.no"], "c-dev")
	}
}
