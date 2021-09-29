package database

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/navikt/datakatalogen/backend/database/gensql"
	"github.com/navikt/datakatalogen/backend/openapi"
)

func TestRepo(t *testing.T) {
	db, err := sql.Open("postgres", "user=postgres dbname=datakatalogen sslmode=disable password=navikt")
	if err != nil {
		t.Fatal(err)
	}
	repo, _ := New(gensql.New(db))

	res, err := repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
		Name: "Hello",
		Owner: openapi.Owner{
			Team: "asdf",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("NEW ID:", res.Id)

	getRes, err := repo.GetDataproduct(context.Background(), res.Id)
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != getRes.Name {
		t.Fatal("navnene er ikke like :/")
	}

	_, err = repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
		Name: "Hello again",
		Owner: openapi.Owner{
			Team: "asdf",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	allRes, err := repo.GetDataproducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(allRes) < 1 {
		t.Fatal("ingen dataprodukter i databasen :thinking:")
	}

	desc := "best description"
	_, err = repo.UpdateDataproduct(context.Background(), res.Id, openapi.NewDataproduct{
		Name:        res.Name,
		Description: &desc,
		Owner:       res.Owner,
		Keyword:     res.Keyword,
		Repo:        res.Repo,
	})
	if err != nil {
		t.Fatal(err)
	}

	updatedRes, err := repo.GetDataproduct(context.Background(), res.Id)
	if err != nil {
		t.Fatal(err)
	}
	if *updatedRes.Description != desc {
		t.Fatal("desc ble ikke oppdatert")
	}

	if err = repo.DeleteDataproduct(context.Background(), res.Id); err != nil {
		t.Fatal(err)
	}
}
