//go:build integration_test

package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
)

var dbString string

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

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestRepo(t *testing.T) {
	repo, err := New(dbString, &event.Manager{}, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		t.Fatal(err)
	}

	newDataproduct := models.NewDataproduct{
		Name: "test-product",
		BigQuery: models.NewBigQuery{
			Dataset:   "dataset",
			ProjectID: "projectid",
			Table:     "table",
		},
		Group: auth.MockUser.Groups[0].Name,
	}

	t.Run("updates dataproducts", func(t *testing.T) {
		data := models.NewDataproduct{
			Name: "test-product",
			BigQuery: models.NewBigQuery{
				Dataset:   "dataset",
				ProjectID: "projectid",
				Table:     "table",
			},
		}
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), data)
		if err != nil {
			t.Fatal(err)
		}

		updated := models.UpdateDataproduct{
			Name: "updated",
			Pii:  false,
		}
		if err != nil {
			t.Fatal(err)
		}

		updatedDataproduct, err := repo.UpdateDataproduct(context.Background(), createdDataproduct.ID, updated)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDataproduct.ID != createdDataproduct.ID {
			t.Fatal("updating dataproduct should not alter dataproduct ID")
		}

		if updatedDataproduct.Name != updated.Name {
			t.Fatal("returned name should match updated name")
		}
	})

	t.Run("deletes dataproducts", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.DeleteDataproduct(context.Background(), createdDataproduct.ID); err != nil {
			t.Fatal(err)
		}

		dataproduct, err := repo.GetDataproduct(context.Background(), createdDataproduct.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}

		if dataproduct != nil {
			t.Fatal("dataproduct should not exist")
		}
	})

	t.Run("handles access grants", func(t *testing.T) {
		dpWithUserAccess := func(ti time.Time, subj string) {
			dp, err := repo.CreateDataproduct(context.Background(), newDataproduct)
			if err != nil {
				t.Fatal(err)
			}
			_, err = repo.GrantAccessToDataproduct(context.Background(), dp.ID, &ti, subj, "")
			if err != nil {
				t.Fatal(err)
			}
		}

		subject := "a@b.com"
		expired := time.Now().Add(-1 * time.Hour)
		valid := time.Now().Add(1 * time.Hour)

		dpWithUserAccess(valid, subject)
		dpWithUserAccess(valid, subject)
		dpWithUserAccess(expired, subject)

		dps, err := repo.GetDataproductsByUserAccess(context.Background(), subject)
		if err != nil {
			t.Fatal(err)
		}

		if len(dps) != 2 {
			t.Errorf("got: %v, want: %v", len(dps), 2)
		}
	})

	t.Run("search datasets and products", func(t *testing.T) {
		tests := map[string]struct {
			query      models.SearchQuery
			numResults int
		}{
			"empty":         {query: models.SearchQuery{Text: stringToPtr("nonexistent")}, numResults: 0},
			"1 dataproduct": {query: models.SearchQuery{Text: stringToPtr("uniquedataproduct")}, numResults: 1},
			"1 results":     {query: models.SearchQuery{Text: stringToPtr("uniquestring")}, numResults: 1},
		}

		dataproduct := models.NewDataproduct{
			Name:        "new_dataproduct",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniquedataproduct"}),
			Pii:         false,
			BigQuery: models.NewBigQuery{
				ProjectID: "project",
				Dataset:   "dataset",
				Table:     "table",
			},
		}

		_, err = repo.CreateDataproduct(context.Background(), dataproduct)
		if err != nil {
			t.Fatal(err)
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				results, err := repo.Search(context.Background(), &tc.query)
				if err != nil {
					t.Fatal(err)
				}

				if len(results) != tc.numResults {
					t.Errorf("expected %v results, got %v", tc.numResults, len(results))
				}
			})
		}
	})
}
