package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/sirupsen/logrus"
)

type DatasetEnricher struct {
	datacatalogClient *Datacatalog
	repo              *database.Repo
	log               *logrus.Entry
}

func New(datacatalogClient *Datacatalog, repo *database.Repo, log *logrus.Entry) *DatasetEnricher {
	return &DatasetEnricher{
		datacatalogClient: datacatalogClient,
		repo:              repo,
		log:               log,
	}
}

func (d *DatasetEnricher) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		if err := d.syncMetadata(ctx); err != nil {
			d.log.WithError(err).Error("Syncing metadata")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			continue
		}
	}
}

func (d *DatasetEnricher) syncMetadata(ctx context.Context) error {
	datasets, err := d.repo.GetDatasets(ctx, math.MaxInt32, 0)
	if err != nil {
		return fmt.Errorf("getting datasets: %w", err)
	}

	for _, ds := range datasets {
		schema, err := d.datacatalogClient.GetDatasetSchema(ctx, ds.Bigquery)
		if err != nil {
			return fmt.Errorf("getting dataset schema: %w", err)
		}

		schemaJSON, err := json.Marshal(schema.Columns)
		if err != nil {
			return fmt.Errorf("marshalling schema: %w", err)
		}

		if err := d.repo.WriteDatasetMetadata(ctx, ds.Id, schemaJSON); err != nil {
			return fmt.Errorf("writing dataset schema to database: %w", err)
		}
	}
	return nil
}
