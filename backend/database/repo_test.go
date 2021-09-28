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
}
