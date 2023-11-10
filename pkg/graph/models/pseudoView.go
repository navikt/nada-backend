package models

import (
	"time"

	"github.com/google/uuid"
)

// NewJoinableViews contains metadata for creating joinable views
type NewJoinableViews struct {
	// Name is the name of the joinable views which will be used as the name of the dataset in bigquery, which contains all the joinable views
	Name    string     `json:"name"`
	Expires *time.Time `json:"expires"`
	// DatasetIDs is the IDs of the datasets which are made joinable.
	DatasetIDs []uuid.UUID `json:"datasetIDs"`
}

type JoinableView struct {
	// id is the id of the joinable view set
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Created          string     `json:"created"`
	Expires          *time.Time `json:"expires"`
	BigQueryViewUrls []string   `json:"bigqueryViewUrls"`
}
