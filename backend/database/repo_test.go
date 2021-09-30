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

	"github.com/navikt/datakatalogen/backend/database/gensql"
	"github.com/navikt/datakatalogen/backend/openapi"
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

	//_, err = repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
	//	Name: "Hello again",
	//	Owner: openapi.Owner{
	//		Team: "asdf",
	//	},
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//allRes, err := repo.GetDataproducts(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//}
	//if len(allRes) < 1 {
	//	t.Fatal("ingen dataprodukter i databasen :thinking:")
	//}
	//
	//desc := "best description"
	//_, err = repo.UpdateDataproduct(context.Background(), res.Id, openapi.NewDataproduct{
	//	Name:        res.Name,
	//	Description: &desc,
	//	Owner:       res.Owner,
	//	Keywords:    res.Keywords,
	//	Repo:        res.Repo,
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//updatedRes, err := repo.GetDataproduct(context.Background(), res.Id)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//if *updatedRes.Description != desc + "banan" {
	//	t.Fatal("desc ble ikke oppdatert")
	//}
	//
	//createdDataset, err := repo.CreateDataset(context.Background(), openapi.NewDataset{
	//	Name:        "My dataset",
	//	DataproductId: updatedRes.Id,
	//	Description: stringToPtr("This is my dataset"),
	//	Pii:         false,
	//	Bigquery: openapi.BigQuery{
	//		ProjectId: "dataplattform-dev-9da3",
	//		Dataset:   "ereg",
	//		Table:     "ereg",
	//	},
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//fetchedDataset, err := repo.GetDataset(context.Background(), createdDataset.Id)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//if createdDataset.Name != fetchedDataset.Name {
	//	t.Fatal("names do not match")
	//}
	//
	//updatedDataset, err := repo.UpdateDataset(context.Background(), createdDataset.Id, openapi.NewDataset{
	//	Name:        "My updated dataset",
	//	DataproductId: updatedRes.Id,
	//	Description: stringToPtr("This is my updated dataset"),
	//	Pii:         false,
	//	Bigquery: openapi.BigQuery{
	//		ProjectId: "dataplattform-dev-9da3",
	//		Dataset:   "ereg",
	//		Table:     "ereg",
	//	},
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//newFetchedDataset, err := repo.GetDataset(context.Background(), createdDataset.Id)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//if updatedDataset.Name != newFetchedDataset.Name {
	//	t.Fatal("names do not match after updating dataset")
	//}
	//
	//if err := repo.DeleteDataset(context.Background(), fetchedDataset.Id); err != nil {
	//	t.Fatal(err)
	//}
	//
	//if err = repo.DeleteDataproduct(context.Background(), res.Id); err != nil {
	//	t.Fatal(err)
	//}
}
