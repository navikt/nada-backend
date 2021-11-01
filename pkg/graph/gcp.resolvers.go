package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) GcpGetTables(ctx context.Context, projectID string, datasetID string) ([]*models.BigQueryTable, error) {
	return r.bigquery.GetTables(ctx, projectID, datasetID)
}

func (r *queryResolver) GcpGetDatasets(ctx context.Context, projectID string) ([]string, error) {
	return r.bigquery.GetDatasets(ctx, projectID)
}
