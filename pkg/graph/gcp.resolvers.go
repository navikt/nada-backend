package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

// GcpGetTables is the resolver for the gcpGetTables field.
func (r *queryResolver) GcpGetTables(ctx context.Context, projectID string, datasetID string) ([]*models.BigQueryTable, error) {
	return r.bigquery.GetTables(ctx, projectID, datasetID)
}

// GcpGetDatasets is the resolver for the gcpGetDatasets field.
func (r *queryResolver) GcpGetDatasets(ctx context.Context, projectID string) ([]string, error) {
	return r.bigquery.GetDatasets(ctx, projectID)
}

// GcpGetAllTablesInProject is the resolver for the gcpGetAllTablesInProject field.
func (r *queryResolver) GcpGetAllTablesInProject(ctx context.Context, projectID string) ([]*models.BigQuerySource, error) {
	datasets, err := r.bigquery.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tables := []*models.BigQuerySource{}
	for _, dataset := range datasets {
		dsTables, err := r.bigquery.GetTables(ctx, projectID, dataset)
		if err != nil {
			return nil, err
		}

		for _, t := range dsTables {
			tables = append(tables, &models.BigQuerySource{
				Table:   t.Name,
				Dataset: dataset,
			})
		}
	}

	return tables, nil
}
