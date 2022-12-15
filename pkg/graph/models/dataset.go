package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type Dataset struct {
	ID                       uuid.UUID `json:"id"`
	DataproductID            uuid.UUID `json:"dataproductID"`
	Name                     string    `json:"name"`
	Created                  time.Time `json:"created"`
	LastModified             time.Time `json:"lastModified"`
	Description              *string   `json:"description"`
	Slug                     string    `json:"slug"`
	Repo                     *string   `json:"repo"`
	Pii                      PiiLevel  `json:"pii"`
	Keywords                 []string  `json:"keywords"`
	Type                     gensql.DatasourceType
	AnonymisationDescription *string `json:"anonymisationDescription"`
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
	PiiTags      *string      `json:"piiTags"`
}

func (BigQuery) IsDatasource() {}

type NewBigQuery struct {
	ProjectID string  `json:"projectID"`
	Dataset   string  `json:"dataset"`
	Table     string  `json:"table"`
	PiiTags   *string `json:"piiTags"`
}

type NewDataset struct {
	DataproductID            uuid.UUID   `json:"dataproductID"`
	Name                     string      `json:"name"`
	Description              *string     `json:"description"`
	Slug                     *string     `json:"slug"`
	Repo                     *string     `json:"repo"`
	Pii                      PiiLevel    `json:"pii"`
	Keywords                 []string    `json:"keywords"`
	BigQuery                 NewBigQuery `json:"bigquery"`
	Requesters               []string    `json:"requesters"`
	AnonymisationDescription *string     `json:"anonymisationDescription"`
	GrantAllUsers            *bool       `json:"grantAllUsers"`
	Metadata                 BigqueryMetadata
}

// NewDatasetForNewDataproduct contains metadata for creating a new dataset for a new dataproduct
type NewDatasetForNewDataproduct struct {
	Name                     string      `json:"name"`
	Description              *string     `json:"description"`
	Repo                     *string     `json:"repo"`
	Pii                      PiiLevel    `json:"pii"`
	Keywords                 []string    `json:"keywords"`
	Bigquery                 NewBigQuery `json:"bigquery"`
	Requesters               []string    `json:"requesters"`
	AnonymisationDescription *string     `json:"anonymisationDescription"`
	GrantAllUsers            *bool       `json:"grantAllUsers"`
	Metadata                 BigqueryMetadata
}

type UpdateDataset struct {
	Name                     string     `json:"name"`
	Description              *string    `json:"description"`
	Slug                     *string    `json:"slug"`
	Repo                     *string    `json:"repo"`
	Pii                      PiiLevel   `json:"pii"`
	Keywords                 []string   `json:"keywords"`
	Requesters               []string   `json:"requesters"`
	DataproductID            *uuid.UUID `json:"dataproductID"`
	AnonymisationDescription *string    `json:"anonymisationDescription"`
	PiiTags                  *string    `json:"piiTags"`
}

type DatasetServices struct {
	Metabase *string `json:"metabase"`
}
