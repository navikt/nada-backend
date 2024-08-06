package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/googleapi"
)

type bigQueryService struct {
	bigQueryStorage    service.BigQueryStorage
	dataProductStorage service.DataProductsStorage
	bigQueryAPI        service.BigQueryAPI
}

var _ service.BigQueryService = &bigQueryService{}

func (s *bigQueryService) GetBigQueryTables(ctx context.Context, projectID string, datasetID string) (*service.BQTables, error) {
	const op errs.Op = "bigQueryService.GetBigQueryTables"

	tables, err := s.bigQueryAPI.GetTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &service.BQTables{
		BQTables: tables,
	}, nil
}

func (s *bigQueryService) GetBigQueryDatasets(ctx context.Context, projectID string) (*service.BQDatasets, error) {
	const op errs.Op = "bigQueryService.GetBigQueryDatasets"

	datasets, err := s.bigQueryAPI.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &service.BQDatasets{
		BQDatasets: datasets,
	}, nil
}

func (s *bigQueryService) GetBigQueryColumns(ctx context.Context, projectID string, datasetID string, tableID string) (*service.BQColumns, error) {
	const op errs.Op = "bigQueryService.GetBigQueryColumns"

	metadata, err := s.bigQueryAPI.TableMetadata(ctx, projectID, datasetID, tableID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &service.BQColumns{
		BQColumns: metadata.Schema.Columns,
	}, nil
}

func (s *bigQueryService) UpdateMetadata(ctx context.Context, ds *service.BigQuery) error {
	const op errs.Op = "bigQueryService.UpdateMetadata"

	metadata, err := s.bigQueryAPI.TableMetadata(ctx, ds.ProjectID, ds.Dataset, ds.Table)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.bigQueryStorage.UpdateBigqueryDatasourceSchema(ctx, ds.DatasetID, metadata)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *bigQueryService) SyncBigQueryTables(ctx context.Context) error {
	const op errs.Op = "bigQueryService.SyncBigQueryTables"

	bqs, err := s.bigQueryStorage.GetBigqueryDatasources(ctx)
	if err != nil {
		return errs.E(op, err)
	}

	var errList []error

	for _, bq := range bqs {
		err := s.UpdateMetadata(ctx, bq)
		if err != nil {
			errList = s.handleSyncError(ctx, errList, err, bq)
		}
	}

	// FIXME: not very nice, should probably log all the errors and return something more generic here
	if len(errList) != 0 {
		errMessage := fmt.Sprintf("syncing bigquery tables: %v", errList)
		return errs.E(errs.IO, op, fmt.Errorf("%w", errors.New(errMessage)))
	}

	return nil
}

func (s *bigQueryService) handleSyncError(ctx context.Context, errs []error, err error, bq *service.BigQuery) []error {
	var e *googleapi.Error

	if ok := errors.As(err, &e); ok {
		if e.Code == http.StatusNotFound {
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

func NewBigQueryService(
	bigQueryStorage service.BigQueryStorage,
	bigQueryAPI service.BigQueryAPI,
	dataProductStorage service.DataProductsStorage,
) *bigQueryService {
	return &bigQueryService{
		bigQueryStorage:    bigQueryStorage,
		bigQueryAPI:        bigQueryAPI,
		dataProductStorage: dataProductStorage,
	}
}
