package metadata

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/database"
	"math"
)

type DatasetEnricher struct {
	datacatalogClient Client
	repo              *database.Repo
}

func (d *DatasetEnricher) SyncMetadata(ctx context.Context) error {
	datasets, err := d.repo.GetDatasets(ctx, math.MaxInt32, 0)
	if err != nil {
		return fmt.Errorf("getting datasets: %w", err)
	}

	fmt.Println("datasets:", datasets)

	for _, ds := range datasets {
		schema, err := d.datacatalogClient.GetDatasetSchema(ds.Bigquery)
		if err != nil {
			return fmt.Errorf("getting dataset schema: %w", err)
		}
		fmt.Println(schema)
		// write to database
	}
	return nil
}
