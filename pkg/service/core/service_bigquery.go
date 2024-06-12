package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/googleapi"
	"time"
)

type bigQueryService struct {
	bigQueryStorage    service.BigQueryStorage
	dataProductStorage service.DataProductsStorage
	bigQueryAPI        service.BigQueryAPI
}

var _ service.BigQueryService = &bigQueryService{}

func (s *bigQueryService) GetBigQueryTables(ctx context.Context, projectID string, datasetID string) (*service.BQTables, error) {
	tables, err := s.bigQueryAPI.GetTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, fmt.Errorf("getting tables: %w", err)
	}

	return &service.BQTables{
		BQTables: tables,
	}, nil
}

func (s *bigQueryService) GetBigQueryDatasets(ctx context.Context, projectID string) (*service.BQDatasets, error) {
	datasets, err := s.bigQueryAPI.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("getting datasets: %w", err)
	}

	return &service.BQDatasets{
		BQDatasets: datasets,
	}, nil
}

func (s *bigQueryService) GetBigQueryColumns(ctx context.Context, projectID string, datasetID string, tableID string) (*service.BQColumns, error) {
	metadata, err := s.bigQueryAPI.TableMetadata(ctx, projectID, datasetID, tableID)
	if err != nil {
		return nil, fmt.Errorf("getting table metadata: %w", err)
	}

	columns := make([]*service.BigqueryColumn, 0)
	for _, column := range metadata.Schema.Columns {
		columns = append(columns, &service.BigqueryColumn{
			Name:        column.Name,
			Description: column.Description,
			Mode:        column.Mode,
			Type:        column.Type,
		})
	}

	return &service.BQColumns{
		BQColumns: columns,
	}, nil
}

func (s *bigQueryService) UpdateMetadata(ctx context.Context, ds *service.BigQuery) error {
	metadata, err := s.bigQueryAPI.TableMetadata(ctx, ds.ProjectID, ds.Dataset, ds.Table)
	if err != nil {
		return fmt.Errorf("getting dataset schema: %w", err)
	}

	err = s.bigQueryStorage.UpdateBigqueryDatasourceSchema(ctx, ds.DatasetID, metadata)
	if err != nil {
		return fmt.Errorf("updating bigquery datasource schema: %w", err)
	}

	return nil
}

func (s *bigQueryService) SyncBigQueryTables(ctx context.Context) error {
	bqs, err := s.bigQueryStorage.GetBigqueryDatasources(ctx)
	if err != nil {
		return err
	}

	var errs []error

	for _, bq := range bqs {
		err := s.UpdateMetadata(ctx, bq)
		if err != nil {
			errs = s.handleSyncError(ctx, errs, err, bq)
		}
	}

	// FIXME: not very nice
	if len(errs) != 0 {
		errMessage := fmt.Sprintf("syncing bigquery tables: %v", errs)
		return fmt.Errorf("%w", errors.New(errMessage))
	}

	return nil
}

func (s *bigQueryService) handleSyncError(ctx context.Context, errs []error, err error, bq *service.BigQuery) []error {
	var e *googleapi.Error

	if ok := errors.As(err, &e); ok {
		if e.Code == 404 {
			if err := s.handleTableNotFound(ctx, bq); err != nil {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}

	return errs
}

const (
	removalTime = -168 * time.Hour // 1 week
)

func (s *bigQueryService) handleTableNotFound(ctx context.Context, bq *service.BigQuery) error {
	if bq.MissingSince == nil {
		return s.bigQueryStorage.UpdateBigqueryDatasourceMissing(ctx, bq.DatasetID)
	} else if bq.MissingSince.Before(time.Now().Add(removalTime)) {
		return s.dataProductStorage.DeleteDataset(ctx, bq.DatasetID)
	}

	return nil
}

func NewBigQueryService(storage service.BigQueryStorage, api service.BigQueryAPI) *bigQueryService {
	return &bigQueryService{
		bigQueryStorage: storage,
		bigQueryAPI:     api,
	}
}
