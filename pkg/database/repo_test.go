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

	"github.com/google/go-cmp/cmp"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/openapi"
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
	repo, err := New(dbString)
	if err != nil {
		t.Fatal(err)
	}

	newCollection := openapi.NewCollection{
		Name: "new_dataproduct",
		Owner: openapi.Owner{
			Group: "team",
		},
	}

	newDataproduct := openapi.NewDataproduct{
		Name: "test-product",
		Datasource: openapi.Bigquery{
			Dataset:   "dataset",
			ProjectId: "projectid",
			Table:     "table",
		},
		Owner: openapi.Owner{
			Group: auth.MockUser.Groups[0].Name,
		},
	}

	t.Run("creates collections", func(t *testing.T) {
		collection, err := repo.CreateCollection(context.Background(), newCollection)
		if err != nil {
			t.Fatal(err)
		}
		if collection.Id == "" {
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

		colElem := openapi.CollectionElement{
			ElementId:   dataproduct.Id,
			ElementType: openapi.CollectionElementTypeDataproduct,
		}

		if err := repo.AddToCollection(context.Background(), collection.Id, colElem); err != nil {
			t.Fatal(err)
		}

		fetchedCollection, err := repo.GetCollection(context.Background(), collection.Id)
		if err != nil {
			t.Fatal(err)
		}
		if newCollection.Name != fetchedCollection.Name {
			t.Fatal("fetched name should match provided name")
		}

		expected := []openapi.CollectionElement{colElem, colElem}
		if !cmp.Equal(fetchedCollection.Elements, expected) {
			t.Error(cmp.Diff(fetchedCollection.Elements, expected))
		}
	})

	t.Run("create and retrieve dataproduct", func(t *testing.T) {
		data := newDataproduct
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), data)
		if err != nil {
			t.Fatal(err)
		}

		dataproduct, err := repo.GetDataproduct(context.Background(), createdDataproduct.Id)
		if err != nil {
			t.Fatal(err)
		}

		if dataproduct.Datasource == nil {
			t.Fatal("Expected datasource to be set")
		}

		bq, ok := dataproduct.Datasource.(openapi.Bigquery)
		if !ok {
			t.Fatalf("Expected datasource to be openapi.Bigquery, got %T", dataproduct.Datasource)
		}

		if !cmp.Equal(bq, data.Datasource) {
			t.Error(cmp.Diff(bq, data.Datasource))
		}
	})

	t.Run("updates dataproducts", func(t *testing.T) {
		data := openapi.NewDataproduct{
			Name: "test-product",
			Datasource: openapi.Bigquery{
				Dataset:   "dataset",
				ProjectId: "projectid",
				Table:     "table",
			},
		}
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), data)
		if err != nil {
			t.Fatal(err)
		}

		updated := openapi.UpdateDataproduct{
			Name: "updated",
			Pii:  false,
		}
		if err != nil {
			t.Fatal(err)
		}

		updatedDataproduct, err := repo.UpdateDataproduct(context.Background(), createdDataproduct.Id, updated)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDataproduct.Id != createdDataproduct.Id {
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

		if err := repo.DeleteDataproduct(context.Background(), createdDataproduct.Id); err != nil {
			t.Fatal(err)
		}

		dataproduct, err := repo.GetDataproduct(context.Background(), createdDataproduct.Id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}

		if dataproduct != nil {
			t.Fatal("dataproduct should not exist")
		}
	})

	t.Run("search datasets and products", func(t *testing.T) {
		tests := map[string]struct {
			query      string
			numResults int
		}{
			"empty":            {query: "nonexistent", numResults: 0},
			"1 dataproduct":    {query: "uniquedataproduct", numResults: 1},
			"1 datacollection": {query: "uniquecollection", numResults: 1},
			"2 results":        {query: "uniquestring", numResults: 2},
		}

		dataproduct := openapi.NewDataproduct{
			Name:        "new_dataproduct",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniquedataproduct"}),
			Pii:         false,
			Datasource: openapi.Bigquery{
				ProjectId: "project",
				Dataset:   "dataset",
				Table:     "table",
			},
		}

		_, err = repo.CreateDataproduct(context.Background(), dataproduct)
		if err != nil {
			t.Fatal(err)
		}

		collection := openapi.NewCollection{
			Name:        "new_collection",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniquecollection"}),
		}

		_, err = repo.CreateCollection(context.Background(), collection)
		if err != nil {
			t.Fatal(err)
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				results, err := repo.Search(context.Background(), tc.query, 15, 0)
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
