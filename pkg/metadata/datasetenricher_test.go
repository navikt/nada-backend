package metadata

import (
	"context"
	"github.com/navikt/nada-backend/pkg/database"
	"testing"
)

func TestName(t *testing.T) {
	repo, err := database.New("postgres://postgres:postgres@127.0.0.1:5432/nada?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	de := DatasetEnricher{
		datacatalogClient: Datacatalog{},
		repo:              repo,
	}

	err = de.SyncMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}
