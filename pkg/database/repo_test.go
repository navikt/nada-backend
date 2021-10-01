//go:build integration_test
// +build integration_test

package database

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func TestRepo(t *testing.T) {
	db, err := sql.Open("postgres", "user=postgres dbname=datakatalogen sslmode=disable password=postgres port=5433")
	if err != nil {
		t.Fatal(err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	repo, _ := New(gensql.New(db))

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

		if dataset != nil {
			t.Fatal("dataset should not exist")
		}
	})
}
