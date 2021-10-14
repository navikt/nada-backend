package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"time"

	"github.com/sirupsen/logrus"
)

type DatasetEnricher struct {
	datacatalogClient Datacataloger
	repo              Metadatastorer
	log               *logrus.Entry
}

type errorList []error

func (e errorList) Error() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", []error(e))
}

type Datacataloger interface {
	GetDatasetSchema(ctx context.Context, ds gensql.DatasourceBigquery) (Schema, error)
}

type Metadatastorer interface {
	GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error)
	UpdateBigqueryDatasource(ctx context.Context, id uuid.UUID, schema json.RawMessage) error
}

func New(datacatalogClient Datacataloger, repo Metadatastorer, log *logrus.Entry) *DatasetEnricher {
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
			el := errorList{}
			if errors.As(err, &el) {
				for _, err := range el {
					d.log.WithError(err).Error("Syncing metadata")
				}
			} else {
				d.log.WithError(err).Error("Syncing metadata")
			}
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
	datasets, err := d.repo.GetBigqueryDatasources(ctx)
	if err != nil {
		return fmt.Errorf("getting datasets: %w", err)
	}

	var errs errorList

	for _, ds := range datasets {
		schema, err := d.datacatalogClient.GetDatasetSchema(ctx, ds)
		if err != nil {
			errs = append(errs, fmt.Errorf("getting dataset schema: %w", err))
			continue
		}

		schemaJSON, err := json.Marshal(schema.Columns)
		if err != nil {
			errs = append(errs, fmt.Errorf("marshalling schema: %w", err))
			continue
		}

		if err := d.repo.UpdateBigqueryDatasource(ctx, ds.DataproductID, schemaJSON); err != nil {
			errs = append(errs, fmt.Errorf("writing metadata to database: %w", err))
			continue
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}