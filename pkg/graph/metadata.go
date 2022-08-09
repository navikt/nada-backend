package graph

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/navikt/nada-backend/pkg/database/gensql"
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
