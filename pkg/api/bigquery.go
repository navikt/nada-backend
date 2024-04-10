package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
)

type GCPProject struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Group *auth.Group `json:"group"`
}

type BigQuery struct {
	ID            uuid.UUID
	DatasetID     uuid.UUID
	ProjectID     string                     `json:"projectID"`
	Dataset       string                     `json:"dataset"`
	Table         string                     `json:"table"`
	TableType     bqclient.BigQueryType      `json:"tableType"`
	LastModified  time.Time                  `json:"lastModified"`
	Created       time.Time                  `json:"created"`
	Expires       *time.Time                 `json:"expired"`
	Description   string                     `json:"description"`
	PiiTags       *string                    `json:"piiTags"`
	MissingSince  *time.Time                 `json:"missingSince"`
	PseudoColumns []string                   `json:"pseudoColumns"`
	Schema        []*bqclient.BigqueryColumn `json:"schema"`
}

type BQTables struct {
	BQTables []*bqclient.BigQueryTable `json:"bqTables"`
}

type BQDatasets struct {
	BQDatasets []string `json:"bqDatasets"`
}

type BQColumns struct {
	BQColumns []*bqclient.BigqueryColumn `json:"bqColumns"`
}

func getBQTables(ctx context.Context, projectID string, datasetID string) (*BQTables, *APIError) {
	tables, err := bqclient.GetTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to retrive bigquery tables")
	}
	return &BQTables{BQTables: tables}, nil
}

func getBQDatasets(ctx context.Context, projectID string) (*BQDatasets, *APIError) {
	datasets, err := bqclient.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to retrive bigquery datasets")
	}
	return &BQDatasets{
		BQDatasets: datasets,
	}, nil
}

func getBQColumns(ctx context.Context, projectID string, datasetID string, tableID string) (*BQColumns, *APIError) {
	metadata, err := bqclient.TableMetadata(ctx, projectID, datasetID, tableID)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to retrive bigquery table metadata")
	}

	columns := make([]*bqclient.BigqueryColumn, 0)
	for _, column := range metadata.Schema.Columns {
		columns = append(columns, &bqclient.BigqueryColumn{
			Name:        column.Name,
			Description: column.Description,
			Mode:        column.Mode,
			Type:        column.Type,
		})
	}
	return &BQColumns{
		BQColumns: columns,
	}, nil
}
