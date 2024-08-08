package service

import (
	"time"

	"github.com/google/uuid"
)

type DatasetInDataproduct struct {
	ID                     uuid.UUID `json:"id"`
	DataproductID          uuid.UUID `json:"-"`
	Name                   string    `json:"name"`
	Created                time.Time `json:"created"`
	LastModified           time.Time `json:"lastModified"`
	Description            *string   `json:"description"`
	Slug                   string    `json:"slug"`
	Keywords               []string  `json:"keywords"`
	DataSourceLastModified time.Time `json:"dataSourceLastModified"`
}

type DataproductOwner struct {
	Group            string     `json:"group"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	TeamContact      *string    `json:"teamContact"`
	TeamID           *string    `json:"teamID"`
	ProductAreaID    *uuid.UUID `json:"productAreaID"`
}

type Dataproduct struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Created         time.Time         `json:"created"`
	LastModified    time.Time         `json:"lastModified"`
	Description     *string           `json:"description"`
	Slug            string            `json:"slug"`
	Owner           *DataproductOwner `json:"owner"`
	Keywords        []string          `json:"keywords"`
	TeamName        *string           `json:"teamName"`
	ProductAreaName string            `json:"productAreaName"`
}

type DataproductWithDataset struct {
	Dataproduct
	Datasets []*DatasetInDataproduct `json:"datasets"`
}
