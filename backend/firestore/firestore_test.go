package firestore_test

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"testing"

	"github.com/navikt/datakatalogen/backend/firestore"
	"github.com/stretchr/testify/assert"
)

func TestFirestoreCRUD(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	f, err := firestore.New(ctx, "aura-dev-d9f5", "dp", "au")
	assert.NoError(t, err)

	clean(t, f)

	t.Run("Create", func(t *testing.T) {
		testDataproduct := firestore.Dataproduct{
			Name:        "dp",
			Description: "desc",
			Datastore:   []map[string]string{{}},
			Team:        "team",
			Access:      nil,
		}

		dpID, err := f.CreateDataproduct(ctx, testDataproduct)

		assert.NoError(t, err)

		dataproduct, err := f.GetDataproduct(ctx, dpID)
		assert.Equal(t, testDataproduct.Name, dataproduct.Dataproduct.Name)
	})

	t.Run("Update", func(t *testing.T) {
		dps, err := f.GetDataproducts(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dps))

		dp := dps[0]

		assert.Equal(t, "dp", dp.Dataproduct.Name)
		err = f.UpdateDataproduct(ctx, dp.ID, firestore.Dataproduct{Name: "dpdp"})
		assert.NoError(t, err)

		newDp, err := f.GetDataproduct(ctx, dp.ID)
		assert.NoError(t, err)

		assert.True(t, dp.Updated.Before(newDp.Updated))
		assert.True(t, newDp.Created.Before(newDp.Updated))
		assert.Equal(t, "dpdp", newDp.Dataproduct.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		dps, err := f.GetDataproducts(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dps))

		dp := dps[0]

		err = f.DeleteDataproduct(ctx, dp.ID)
		assert.NoError(t, err)

		dps, err = f.GetDataproducts(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(dps))
	})

	t.Run("Invalid", func(t *testing.T) {
		invalidProduct := firestore.Dataproduct{}
		dpID, err := f.CreateDataproduct(ctx, invalidProduct)

		assert.NoError(t, err)

		dataproduct, err := f.GetDataproduct(ctx, dpID)
		assert.Error(t, err)
		assert.Equal(t, status.Code(err), codes.NotFound)
		assert.Nil(t, dataproduct)

		dataproducts, err := f.GetDataproducts(ctx)
		assert.NoError(t, err)
		assert.Len(t, dataproducts, 0, "invalid dataproducts are skipped")
	})
}

func clean(t *testing.T, f *firestore.Firestore) {
	ctx := context.Background()
	dataproducts, err := f.GetDataproducts(ctx)
	if err != nil {
		t.Fatalf("Getting dataproducts: %v", err)
	}
	for _, dp := range dataproducts {
		if err := f.DeleteDataproduct(ctx, dp.ID); err != nil {
			t.Fatalf("Deleting dataproduct: %v", err)
		}
	}
}
