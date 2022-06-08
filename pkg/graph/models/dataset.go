package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type Dataset struct {
	ID            uuid.UUID `json:"id"`
	DataproductID uuid.UUID `json:"dataproductID"`
	Name          string    `json:"name"`
	Created       time.Time `json:"created"`
	LastModified  time.Time `json:"lastModified"`
	Description   *string   `json:"description"`
	Slug          string    `json:"slug"`
	Repo          *string   `json:"repo"`
	Pii           bool      `json:"pii"`
	Keywords      []string  `json:"keywords"`
	Type          gensql.DatasourceType
}

func (Dataset) IsSearchResult() {}

type Datasource interface {
	IsDatasource()
}

type BigQuery struct {
	DatasetID    uuid.UUID
	ProjectID    string       `json:"projectID"`
	Dataset      string       `json:"dataset"`
	Table        string       `json:"table"`
	TableType    BigQueryType `json:"tableType"`
	LastModified time.Time    `json:"lastModified"`
	Created      time.Time    `json:"created"`
	Expires      *time.Time   `json:"expired"`
	Description  string       `json:"description"`
}

func (BigQuery) IsDatasource() {}

type NewBigQuery struct {
	ProjectID string `json:"projectID"`
	Dataset   string `json:"dataset"`
	Table     string `json:"table"`
}

type NewDataset struct {
	DataproductID uuid.UUID   `json:"dataproductID"`
	Name          string      `json:"name"`
	Description   *string     `json:"description"`
	Slug          *string     `json:"slug"`
	Repo          *string     `json:"repo"`
	Pii           bool        `json:"pii"`
	Keywords      []string    `json:"keywords"`
	BigQuery      NewBigQuery `json:"bigquery"`
	Requesters    []string    `json:"requesters"`
	Metadata      BigqueryMetadata
}

// NewDatasetForNewDataproduct contains metadata for creating a new dataset for a new dataproduct
type NewDatasetForNewDataproduct struct {
	// name of dataset
	Name string `json:"name"`
	// description of the dataset
	Description *string `json:"description"`
	// repo is the url of the repository containing the code to create the dataset
	Repo *string `json:"repo"`
	// pii indicates whether it is personal identifiable information in the dataset
	Pii bool `json:"pii"`
	// keywords for the dataset used as tags.
	Keywords []string `json:"keywords"`
	// bigquery contains metadata for the bigquery datasource added to the dataset.
	Bigquery *NewBigQuery `json:"bigquery"`
	// requesters contains list of users, groups and service accounts which can request access to the dataset
	Requesters []string `json:"requesters"`
}

type UpdateDataset struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Repo        *string  `json:"repo"`
	Pii         bool     `json:"pii"`
	Keywords    []string `json:"keywords"`
	Requesters  []string `json:"requesters"`
}

type DatasetServices struct {
	Metabase *string `json:"metabase"`
}
