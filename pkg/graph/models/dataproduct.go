package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type Dataproduct struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Description  *string   `json:"description"`
	Slug         string    `json:"slug"`
	Repo         *string   `json:"repo"`
	Pii          bool      `json:"pii"`
	Keywords     []string  `json:"keywords"`
	Owner        *Owner    `json:"owner"`
	Type         gensql.DatasourceType
}

func (Dataproduct) IsCollectionElement() {}

type Datasource interface {
	IsDatasource()
}

type BigQuery struct {
	ProjectID string `json:"projectID"`
	Dataset   string `json:"dataset"`
	Table     string `json:"table"`
}

func (BigQuery) IsDatasource() {}

type NewBigQuery struct {
	ProjectID string `json:"projectID"`
	Dataset   string `json:"dataset"`
	Table     string `json:"table"`
}

type NewDataproduct struct {
	Name        string      `json:"name"`
	Description *string     `json:"description"`
	Slug        string      `json:"slug"`
	Repo        *string     `json:"repo"`
	Pii         bool        `json:"pii"`
	Keywords    []string    `json:"keywords"`
	Group       string      `json:"group"`
	BigQuery    NewBigQuery `json:"bigquery"`
}

type Owner struct {
	Group         string `json:"group"`
	Teamkatalogen string `json:"teamkatalogen"`
}
