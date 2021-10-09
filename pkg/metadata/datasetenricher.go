package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/navikt/nada-backend/pkg/database"
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

	for _, ds := range datasets {
		fmt.Println("dataset:", ds)
		schema, err := d.datacatalogClient.GetDatasetSchema(ds.Bigquery)
		if err != nil {
			return fmt.Errorf("getting dataset schema: %w", err)
		}
		fmt.Println(schema.Columns)

		schemaJSON, err := json.Marshal(schema.Columns)
		if err != nil {
			return fmt.Errorf("getting dataset schema: %w", err)
		}

		if err := d.repo.WriteDatasetMetadata(ctx, ds.Id, schemaJSON); err != nil {
			return fmt.Errorf("writing dataset schema to database: %w", err)
		}
	}
	return nil
}
