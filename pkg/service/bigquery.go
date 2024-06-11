package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sqlc-dev/pqtype"
	"google.golang.org/api/googleapi"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

// For now, just exporting the functionality that is needed by Metabase
type BigQueryStorage interface {
	GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID, isReference bool) (*BigQuery, error)
}

// Same here
type BigQueryAPI interface {
	Grant(ctx context.Context, projectID, datasetID, tableID, member string) error
	Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error
	HasAccess(ctx context.Context, projectID, datasetID, tableID, member string) (bool, error)
	AddToAuthorizedViews(ctx context.Context, srcProjectID, srcDataset, sinkProjectID, sinkDataset, sinkTable string) error
}

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

func GetBQTables(ctx context.Context, projectID string, datasetID string) (*BQTables, *APIError) {
	tables, err := bq.GetTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to retrive bigquery tables")
	}
	return &BQTables{BQTables: tables}, nil
}

func GetBQDatasets(ctx context.Context, projectID string) (*BQDatasets, *APIError) {
	datasets, err := bq.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to retrieve bigquery datasets")
	}
	return &BQDatasets{
		BQDatasets: datasets,
	}, nil
}

func GetBQColumns(ctx context.Context, projectID string, datasetID string, tableID string) (*BQColumns, *APIError) {
	metadata, err := bq.TableMetadata(ctx, projectID, datasetID, tableID)
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

func GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID, isReference bool) (*BigQuery, *APIError) {
	bq, err := queries.GetBigqueryDatasource(ctx, gensql.GetBigqueryDatasourceParams{
		DatasetID:   datasetID,
		IsReference: isReference,
	})
	if err == sql.ErrNoRows {
		return nil, NewAPIError(http.StatusNotFound, err, fmt.Sprintf("getBigqueryDatasource(): bigquery datasource not found for %v", datasetID))
	} else if err != nil {
		return nil, DBErrorToAPIError(err, fmt.Sprintf("getBigqueryDatasource(): failed to get bigquery datasource for %v", datasetID))
	}

	piiTags := "{}"
	if bq.PiiTags.RawMessage != nil {
		piiTags = string(bq.PiiTags.RawMessage)
	}

	return &BigQuery{
		ID:            bq.ID,
		DatasetID:     bq.DatasetID,
		ProjectID:     bq.ProjectID,
		Dataset:       bq.Dataset,
		Table:         bq.TableName,
		TableType:     bqclient.BigQueryType(strings.ToLower(bq.TableType)),
		LastModified:  bq.LastModified,
		Created:       bq.Created,
		Expires:       nullTimeToPtr(bq.Expires),
		Description:   bq.Description.String,
		PiiTags:       &piiTags,
		MissingSince:  &bq.MissingSince.Time,
		PseudoColumns: bq.PseudoColumns,
	}, nil
}

func GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, *APIError) {
	bqs, err := queries.GetBigqueryDatasources(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, DBErrorToAPIError(err, fmt.Sprintf("fetching bigquery datasources"))
	}

	return bqs, nil
}

const (
	removalTime = -168 * time.Hour // 1 week
)

type ErrorList []error

func (e ErrorList) Error() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("%+v", []error(e))
}

func UpdateMetadata(ctx context.Context, ds gensql.DatasourceBigquery) error {
	metadata, err := bq.TableMetadata(ctx, ds.ProjectID, ds.Dataset, ds.TableName)
	if err != nil {
		return fmt.Errorf("getting dataset schema: %w", err)
	}

	schemaJSON, err := json.Marshal(metadata.Schema.Columns)
	if err != nil {
		return fmt.Errorf("marshalling schema: %w", err)
	}

	err = queries.UpdateBigqueryDatasourceSchema(ctx, gensql.UpdateBigqueryDatasourceSchemaParams{
		Schema: pqtype.NullRawMessage{
			RawMessage: schemaJSON,
			Valid:      true,
		},
		LastModified:  metadata.LastModified,
		Expires:       sql.NullTime{Time: metadata.Expires, Valid: !metadata.Expires.IsZero()},
		Description:   sql.NullString{String: metadata.Description, Valid: true},
		PseudoColumns: nil,
		DatasetID:     ds.DatasetID,
	})
	if err != nil {
		return fmt.Errorf("writing metadata to database: %w", err)
	}

	return nil
}

func HandleSyncError(ctx context.Context, errs ErrorList, err error, bq gensql.DatasourceBigquery) ErrorList {
	var e *googleapi.Error

	if ok := errors.As(err, &e); ok {
		if e.Code == 404 {
			if err := handleTableNotFound(ctx, bq); err != nil {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}

	return errs
}

func handleTableNotFound(ctx context.Context, bq gensql.DatasourceBigquery) error {
	if !bq.MissingSince.Valid {
		return queries.UpdateBigqueryDatasourceMissing(ctx, bq.DatasetID)
	} else if bq.MissingSince.Time.Before(time.Now().Add(removalTime)) {
		return queries.DeleteDataset(ctx, bq.DatasetID)
	}

	return nil
}
