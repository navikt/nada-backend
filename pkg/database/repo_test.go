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

	newDataproduct := openapi.NewDataproduct{
		Name: "new_dataproduct",
		Owner: openapi.Owner{
			Team: "team",
		},
	}

	t.Run("creates dataproducts", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}
		if createdDataproduct.Id == "" {
			t.Fatal("returned dataproducts should contain ID")
		}
		if newDataproduct.Name != createdDataproduct.Name {
			t.Fatal("returned name should match provided name")
		}
		if len(createdDataproduct.Datasets) > 0 {
			t.Fatal("returned dataproduct datasets should be empty")
		}
	})

	t.Run("serves dataproducts", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		fetchedDataproduct, err := repo.GetDataproduct(context.Background(), createdDataproduct.Id)
		if err != nil {
			t.Fatal(err)
		}
		if newDataproduct.Name != fetchedDataproduct.Name {
			t.Fatal("fetched name should match provided name")
		}
	})

	t.Run("serves dataproducts with dataset", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		newDataset := openapi.NewDataset{
			Name:          "new_dataset",
			DataproductId: createdDataproduct.Id,
			Pii:           false,
			Bigquery: openapi.BigQuery{
				ProjectId: "project",
				Dataset:   "dataset",
				Table:     "table",
			},
		}

		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		dataproducts, err := repo.GetDataproducts(context.Background(), 10, 0)
		if err != nil {
			t.Fatal(err)
		}

		for _, dp := range dataproducts {
			if dp.Id != createdDataproduct.Id {
				continue
			}

			if len(dp.Datasets) == 0 {
				t.Fatal("Expected dataset to be at least of size 1")
			}

			if dp.Datasets[0].Name != createdDataset.Name {
				t.Fatal("Dataset names doesn't match")
			}
		}
	})

	t.Run("serves dataproduct with dataset", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		newDataset := openapi.NewDataset{
			Name:          "new_dataset",
			DataproductId: createdDataproduct.Id,
			Pii:           false,
			Bigquery: openapi.BigQuery{
				ProjectId: "project",
				Dataset:   "dataset",
				Table:     "table",
			},
		}

		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		dataproducts, err := repo.GetDataproduct(context.Background(), createdDataproduct.Id)
		if err != nil {
			t.Fatal(err)
		}

		if len(dataproducts.Datasets) == 0 {
			t.Fatal("Expected dataset to be at least of size 1")
		}

		if dataproducts.Datasets[0].Name != createdDataset.Name {
			t.Fatal("Dataset names doesn't match")
		}
	})

	t.Run("updates dataproducts", func(t *testing.T) {
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		dataproductWithUpdate := newDataproduct
		dataproductWithUpdate.Name = "updated"
		updatedDataproduct, err := repo.UpdateDataproduct(context.Background(), createdDataproduct.Id, dataproductWithUpdate)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDataproduct.Name != dataproductWithUpdate.Name {
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

	createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
	if err != nil {
		t.Fatal(err)
	}

	newDataset := openapi.NewDataset{
		Name:          "new_dataset",
		DataproductId: createdDataproduct.Id,
		Pii:           false,
		Bigquery: openapi.BigQuery{
			ProjectId: "project",
			Dataset:   "dataset",
			Table:     "table",
		},
	}

	t.Run("creates datasets", func(t *testing.T) {
		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		if createdDataset.Id == "" {
			t.Fatal("returned dataset should contain ID")
		}

		if newDataset.Name != createdDataset.Name {
			t.Fatal("returned name should match provided name")
		}
	})

	t.Run("serves datasets", func(t *testing.T) {
		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		fetchedDataset, err := repo.GetDataset(context.Background(), createdDataset.Id)
		if err != nil {
			t.Fatal(err)
		}
		if newDataset.Name != fetchedDataset.Name {
			t.Fatal("fetched name should match provided name")
		}
	})

	t.Run("update datasets", func(t *testing.T) {
		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		datasetWithUpdate := newDataset
		datasetWithUpdate.Name = "updated"
		updatedDataset, err := repo.UpdateDataset(context.Background(), createdDataset.Id, datasetWithUpdate)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDataset.Name != datasetWithUpdate.Name {
			t.Fatal("returned name should match updated name")
		}
	})

	t.Run("deletes datasets", func(t *testing.T) {
		createdDataset, err := repo.CreateDataset(context.Background(), newDataset)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.DeleteDataset(context.Background(), createdDataset.Id); err != nil {
			t.Fatal(err)
		}

		dataset, err := repo.GetDataset(context.Background(), createdDataset.Id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}

		if dataset != nil {
			t.Fatal("dataset should not exist")
		}
	})

	t.Run("search datasets and products", func(t *testing.T) {
		tests := map[string]struct {
			query      string
			numResults int
		}{
			"empty":         {query: "nonexistent", numResults: 0},
			"1 dataproduct": {query: "uniquedataproduct", numResults: 1},
			"1 dataset":     {query: "uniquedataset", numResults: 1},
			"2 results":     {query: "uniquestring", numResults: 2},
		}

		newDataproduct := openapi.NewDataproduct{
			Name:        "new_dataproduct",
			Description: nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniqueDataproduct"}),
			Owner: openapi.Owner{
				Team: "team",
			},
		}
		createdDataproduct, err := repo.CreateDataproduct(context.Background(), newDataproduct)
		if err != nil {
			t.Fatal(err)
		}

		newDataset := openapi.NewDataset{
			Name:          "new_dataset",
			Description:   nullStringToPtr(sql.NullString{Valid: true, String: "Uniquestring uniqueDataset"}),
			DataproductId: createdDataproduct.Id,
			Pii:           false,
			Bigquery: openapi.BigQuery{
				ProjectId: "project",
				Dataset:   "dataset",
				Table:     "table",
			},
		}

		_, err = repo.CreateDataset(context.Background(), newDataset)
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
