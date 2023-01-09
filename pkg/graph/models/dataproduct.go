package models

import (
	"time"

	"github.com/google/uuid"
)

type Dataproduct struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Description  *string   `json:"description"`
	Slug         string    `json:"slug"`
	Owner        *Owner    `json:"owner"`
}

func (Dataproduct) IsSearchResult() {}

type NewDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Group            string   `json:"group"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	TeamContact      *string  `json:"teamContact"`
	Requesters       []string `json:"requesters"`
	Metadata         BigqueryMetadata
	Datasets         []NewDatasetForNewDataproduct `json:"datasets"`
	ProductAreaID    *string                       `json:"productAreaID"`
	TeamID           *string                       `json:"teamID"`
}

type UpdateDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Pii              PiiLevel `json:"pii"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	TeamContact      *string  `json:"teamContact"`
	Requesters       []string `json:"requesters"`
	ProductAreaID    *string  `json:"productAreaID"`
	TeamID           *string  `json:"teamID"`
}

type Keyword struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type GroupStats struct {
	Email        string `json:"email"`
	Dataproducts int    `json:"dataproducts"`
}
