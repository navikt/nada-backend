//go:build integration_test

package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/google/go-cmp/cmp"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/ory/dockertest/v3"
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
	repo, err := New(dbString, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		t.Fatal(err)
	}

	newCollection := models.NewCollection{
		Name:  "new_dataproduct",
		Group: "team",
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

	t.Run("creates collections", func(t *testing.T) {
		collection, err := repo.CreateCollection(context.Background(), newCollection)
		if err != nil {
			t.Fatal(err)
		}
		if collection.ID.String() == "" {
			t.Fatal("returned collection should contain ID")
		}
		if newCollection.Name != collection.Name {
			t.Fatal("returned name should match provided name")
		}
	})

	t.Run("serves collections", func(t *testing.T) {
		collection, err := repo.CreateCollection(context.Background(), newCollection)
		if err != nil {
			t.Fatal(err)
		}

		dataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.AddToCollection(context.Background(), collection.ID, dataproduct.ID, models.CollectionElementTypeDataproduct.String()); err != nil {
			t.Fatal(err)
		}

		fetchedCollection, err := repo.GetCollection(context.Background(), collection.ID)
		if err != nil {
			t.Fatal(err)
		}
		if newCollection.Name != fetchedCollection.Name {
			t.Fatal("fetched name should match provided name")
		}
		fetchedCollectionElements, err := repo.GetCollectionElements(context.Background(), collection.ID)
		if err != nil {
			t.Fatal(err)
		}

		expected := []models.CollectionElement{dataproduct}
		if !cmp.Equal(fetchedCollectionElements, expected) {
			t.Error(cmp.Diff(fetchedCollectionElements, expected))
		}
	})

	t.Run("create and retrieve dataproduct", func(t *testing.T) {
		data := newDataproduct
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), data)
		if err != nil {
			t.Fatal(err)
		}

		dataproduct, err := repo.GetDataproduct(context.Background(), createdDataproduct.ID)
		if err != nil {
			t.Fatal(err)
		}

		if dataproduct.ID.String() == "" {
			t.Fatal("Expected ID to be set")
		}

		bq, err := repo.GetBigqueryDatasource(context.Background(), dataproduct.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(bq, models.BigQuery(data.BigQuery)) {
			t.Error(cmp.Diff(bq, models.BigQuery(data.BigQuery)))
		}
	})

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

	t.Run("search datasets and products", func(t *testing.T) {
		tests := map[string]struct {
			query      models.SearchQuery
			numResults int
		}{
			"empty":            {query: models.SearchQuery{Text: stringToPtr("nonexistent")}, numResults: 0},
			"1 dataproduct":    {query: models.SearchQuery{Text: stringToPtr("uniquedataproduct")}, numResults: 1},
			"1 datacollection": {query: models.SearchQuery{Text: stringToPtr("uniquecollection")}, numResults: 1},
			"2 results":        {query: models.SearchQuery{Text: stringToPtr("uniquestring")}, numResults: 2},
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

		collection := models.NewCollection{
			Name:        "new_collection",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniquecollection"}),
		}

		_, err = repo.CreateCollection(context.Background(), collection)
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
