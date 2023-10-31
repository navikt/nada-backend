//go:build integration_test

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/event"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
)

var dbString string

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
	repo, err := New(dbString, 2, 0, &event.Manager{}, logrus.NewEntry(logrus.StandardLogger()), "nav-central-data-dev-e170")
	if err != nil {
		t.Fatal(err)
	}

	newDataproduct := models.NewDataproduct{
		Name:  "test-product",
		Group: auth.MockUser.GoogleGroups[0].Name,
	}

	user := &auth.User{
		Email: "user@nav.no",
	}

	createdproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct, user)
	if err != nil {
		t.Fatal(err)
	}

	newDataset := models.NewDataset{
		Name: "test-dataset",
		Pii:  models.PiiLevelSensitive,
		BigQuery: models.NewBigQuery{
			Dataset:   "dataset",
			ProjectID: "projectid",
			Table:     "table",
		},
		DataproductID: createdproduct.ID,
	}

	t.Run("updates datasets", func(t *testing.T) {
		data := models.NewDataset{
			Name: "test-dataset",
			Pii:  models.PiiLevelSensitive,
			BigQuery: models.NewBigQuery{
				Dataset:   "dataset",
				ProjectID: "projectid",
				Table:     "table",
			},
			DataproductID: createdproduct.ID,
		}

		createdDataset, err := repo.CreateDataset(context.Background(), data, nil, user)
		if err != nil {
			t.Fatal(err)
		}

		updated := models.UpdateDataset{
			Name:          "updated",
			Pii:           models.PiiLevelAnonymised,
			DataproductID: &createdproduct.ID,
		}

		updatedDataset, err := repo.UpdateDataset(context.Background(), createdDataset.ID, updated)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDataset.ID != createdDataset.ID {
			t.Fatal("updating dataset should not alter dataset ID")
		}

		if updatedDataset.Name != updated.Name {
			t.Fatal("returned name should match updated name")
		}
	})

	t.Run("deletes datasets", func(t *testing.T) {
		createdDataset, err := repo.CreateDataset(context.Background(), newDataset, nil, user)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.DeleteDataset(context.Background(), createdDataset.ID); err != nil {
			t.Fatal(err)
		}

		dataset, err := repo.GetDataset(context.Background(), createdDataset.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}

		if dataset != nil {
			t.Fatal("dataset should not exist")
		}
	})

	t.Run("handles access grants", func(t *testing.T) {
		dpWithUserAccess := func(ti time.Time, subj string) {
			dp, err := repo.CreateDataset(context.Background(), newDataset, nil, user)
			if err != nil {
				t.Fatal(err)
			}

			fmt.Println(dp)
			_, err = repo.GrantAccessToDataset(context.Background(), dp.ID, &ti, subj, "")
			if err != nil {
				t.Fatal(err)
			}
		}

		subjectWithType := "user:a@b.com"
		expired := time.Now().Add(-1 * time.Hour)
		valid := time.Now().Add(1 * time.Hour)

		dpWithUserAccess(valid, subjectWithType)
		dpWithUserAccess(valid, subjectWithType)
		dpWithUserAccess(expired, subjectWithType)

		dps, err := repo.GetDatasetsByUserAccess(context.Background(), subjectWithType)
		if err != nil {
			t.Fatal(err)
		}

		if len(dps) != 2 {
			t.Errorf("got: %v, want: %v", len(dps), 2)
		}
	})

	t.Run("search dataproduct", func(t *testing.T) {
		tests := map[string]struct {
			query      models.SearchQuery
			numResults int
		}{
			"empty":         {query: models.SearchQuery{Text: stringToPtr("nonexistent")}, numResults: 0},
			"1 dataproduct": {query: models.SearchQuery{Text: stringToPtr("uniquedataproduct")}, numResults: 1},
			"1 story":       {query: models.SearchQuery{Text: stringToPtr("uniquestory")}, numResults: 1},
			"2 results":     {query: models.SearchQuery{Text: stringToPtr("uniquestring")}, numResults: 2},
		}

		dataproduct := models.NewDataproduct{
			Name:        "new dataproduct",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniquedataproduct"}),
		}

		ctx := context.Background()

		_, err := repo.CreateDataproduct(ctx, dataproduct, user)
		if err != nil {
			t.Fatal(err)
		}

		draftID, err := createStoryDraft(ctx, repo)
		if err != nil {
			t.Fatal(err)
		}

		_, err = repo.PublishStory(ctx, models.NewStory{
			ID:       draftID,
			Group:    "group@email.com",
			Keywords: []string{},
		})
		if err != nil {
			t.Fatal(err)
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				results, err := repo.Search(ctx, &tc.query)
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

func createStoryDraft(ctx context.Context, repo *Repo) (uuid.UUID, error) {
	headerView := map[string]interface{}{
		"content": "Header",
		"level":   1,
	}
	headerBytes, err := json.Marshal(headerView)
	if err != nil {
		return uuid.UUID{}, err
	}

	mdView := map[string]interface{}{
		"content": "uniquestring uniquestory",
	}
	mdBytes, err := json.Marshal(mdView)
	if err != nil {
		return uuid.UUID{}, err
	}

	draftID, err := repo.CreateStoryDraft(ctx, &models.DBStory{
		Name: "new story",
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
