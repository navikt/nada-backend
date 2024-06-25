package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BigQueryStorage interface {
	GetBigqueryDatasource(ctx context.Context, datasetID uuid.UUID, isReference bool) (*BigQuery, error)
	GetBigqueryDatasources(ctx context.Context) ([]*BigQuery, error)
	UpdateBigqueryDatasourceSchema(ctx context.Context, datasetID uuid.UUID, meta BigqueryMetadata) error
	UpdateBigqueryDatasourceMissing(ctx context.Context, datasetID uuid.UUID) error
	UpdateBigqueryDatasource(ctx context.Context, input BigQueryDataSourceUpdate) error
	GetPseudoDatasourcesToDelete(ctx context.Context) ([]*BigQuery, error)
}

type BigQueryAPI interface {
	Grant(ctx context.Context, projectID, datasetID, tableID, member string) error
	Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error
	AddToAuthorizedViews(ctx context.Context, srcProjectID, srcDataset, sinkProjectID, sinkDataset, sinkTable string) error
	MakeBigQueryUrlForJoinableViews(name, projectID, datasetID, tableID string) string
	CreateJoinableViewsForUser(ctx context.Context, name string, datasources []JoinableViewDatasource) (string, string, map[string]string, error)
	CreateJoinableView(ctx context.Context, joinableDatasetID string, datasource JoinableViewDatasource) (string, error)
	ComposeJoinableViewQuery(plainTable DatasourceForJoinableView, joinableDatasetID string) string
	TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (BigqueryMetadata, error)
	GetTables(ctx context.Context, projectID, datasetID string) ([]*BigQueryTable, error)
	GetDatasets(ctx context.Context, projectID string) ([]string, error)
	CreatePseudonymisedView(ctx context.Context, projectID, datasetID, tableID string, piiColumns []string) (string, string, string, error)
	DeleteJoinableView(ctx context.Context, joinableViewName, refProjectID, refDatasetID, refTableID string) error
	DeletePseudoView(ctx context.Context, pseudoProjectID, pseudoDatasetID, pseudoTableID string) error
	DeleteJoinableDataset(ctx context.Context, datasetID string) error
}

type BigQueryService interface {
	SyncBigQueryTables(ctx context.Context) error
	UpdateMetadata(ctx context.Context, ds *BigQuery) error
	GetBigQueryTables(ctx context.Context, projectID string, datasetID string) (*BQTables, error)
	GetBigQueryDatasets(ctx context.Context, projectID string) (*BQDatasets, error)
	GetBigQueryColumns(ctx context.Context, projectID string, datasetID string, tableID string) (*BQColumns, error)
}

type BigQueryTableType string

const (
	// RegularTable is a regular table.
	RegularTable BigQueryTableType = "TABLE"
	// ViewTable is a table type describing that the table is a logical view.
	// See more information at https://cloud.google.com//docs/views.
	ViewTable BigQueryTableType = "VIEW"
	// ExternalTable is a table type describing that the table is an external
	// table (also known as a federated data source). See more information at
	// https://cloud.google.com/bigquery/external-data-sources.
	ExternalTable BigQueryTableType = "EXTERNAL"
	// MaterializedView represents a managed storage table that's derived from
	// a base table.
	MaterializedView BigQueryTableType = "MATERIALIZED_VIEW"
	// Snapshot represents an immutable point in time snapshot of some other
	// table.
	Snapshot BigQueryTableType = "SNAPSHOT"
)

type DatasourceForJoinableView struct {
	Project       string
	Dataset       string
	Table         string
	PseudoColumns []string
}

type JoinableViewDatasource struct {
	RefDatasource    *DatasourceForJoinableView
	PseudoDatasource *DatasourceForJoinableView
}

type GCPProject struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group *Group `json:"group"`
}

type BigQuery struct {
	ID            uuid.UUID
	DatasetID     uuid.UUID
	ProjectID     string            `json:"projectID"`
	Dataset       string            `json:"dataset"`
	Table         string            `json:"table"`
	TableType     BigQueryTableType `json:"tableType"`
	LastModified  time.Time         `json:"lastModified"`
	Created       time.Time         `json:"created"`
	Expires       *time.Time        `json:"expired"`
	Description   string            `json:"description"`
	PiiTags       *string           `json:"piiTags"`
	MissingSince  *time.Time        `json:"missingSince"`
	PseudoColumns []string          `json:"pseudoColumns"`
	Schema        []*BigqueryColumn `json:"schema"`
}

type BQTables struct {
	BQTables []*BigQueryTable `json:"bqTables"`
}

type BQDatasets struct {
	BQDatasets []string `json:"bqDatasets"`
}

type BQColumns struct {
	BQColumns []*BigqueryColumn `json:"bqColumns"`
}

type NewBigQuery struct {
	ProjectID string  `json:"projectID"`
	Dataset   string  `json:"dataset"`
	Table     string  `json:"table"`
	PiiTags   *string `json:"piiTags"`
}

type BigquerySchema struct {
	Columns []*BigqueryColumn
}

type BigqueryMetadata struct {
	Schema       BigquerySchema    `json:"schema"`
	TableType    BigQueryTableType `json:"tableType"`
	LastModified time.Time         `json:"lastModified"`
	Created      time.Time         `json:"created"`
	Expires      time.Time         `json:"expires"`
	Description  string            `json:"description"`
}

type BigQueryDataSourceUpdate struct {
	PiiTags       *string
	PseudoColumns []string
	DatasetID     uuid.UUID
}

type BigqueryColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	Description string `json:"description"`
}

type BigQueryTable struct {
	Description  string            `json:"description"`
	LastModified time.Time         `json:"lastModified"`
	Name         string            `json:"name"`
	Type         BigQueryTableType `json:"type"`
}
