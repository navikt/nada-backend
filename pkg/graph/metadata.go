package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"golang.org/x/xerrors"
	"google.golang.org/api/googleapi"
)

const (
	removalTime = -168 * time.Hour // 1 week
)

type errorList []error

func (e errorList) Error() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", []error(e))
}

func (r *mutationResolver) UpdateMetadata(ctx context.Context, ds gensql.DatasourceBigquery) error {
	metadata, err := r.bigquery.TableMetadata(ctx, ds.ProjectID, ds.Dataset, ds.TableName)
	if err != nil {
		return fmt.Errorf("getting dataset schema: %w", err)
	}

	schemaJSON, err := json.Marshal(metadata.Schema.Columns)
	if err != nil {
		return fmt.Errorf("marshalling schema: %w", err)
	}

	if err := r.repo.UpdateBigqueryDatasource(ctx, ds.DatasetID, schemaJSON, metadata.LastModified, metadata.Expires, metadata.Description); err != nil {
		return fmt.Errorf("writing metadata to database: %w", err)
	}

	return nil
}

func (r *mutationResolver) handleSyncError(ctx context.Context, errs errorList, err error, bq gensql.DatasourceBigquery) errorList {
	var e *googleapi.Error
	if ok := xerrors.As(err, &e); ok {
		if e.Code == 404 {
			if err := r.handleTableNotFound(ctx, bq); err != nil {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}

	return errs
}

func (r *mutationResolver) handleTableNotFound(ctx context.Context, bq gensql.DatasourceBigquery) error {
	if !bq.MissingSince.Valid {
		return r.repo.UpdateBigqueryDatasourceMissing(ctx, bq.DatasetID)
	} else if bq.MissingSince.Time.Before(time.Now().Add(removalTime)) {
		return r.repo.DeleteDataset(ctx, bq.DatasetID)
	}

	return nil
}
