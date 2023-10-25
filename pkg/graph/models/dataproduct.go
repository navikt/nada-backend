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

// NewDataproduct contains metadata for creating a new dataproduct
type NewDataproduct struct {
	// name of dataproduct
	Name string `json:"name"`
	// description of the dataproduct
	Description *string `json:"description,omitempty"`
	// owner group email for the dataproduct.
	Group string `json:"group"`
	// owner Teamkatalogen URL for the dataproduct.
	TeamkatalogenURL *string `json:"teamkatalogenURL,omitempty"`
	// The contact information of the team who owns the dataproduct, which can be slack channel, slack account, email, and so on.
	TeamContact *string `json:"teamContact,omitempty"`
	// Id of the team's product area.
	ProductAreaID *string `json:"productAreaID,omitempty"`
	// Id of the team.
	TeamID *string `json:"teamID,omitempty"`
	Slug   *string
}

type UpdateDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Pii              PiiLevel `json:"pii"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	TeamContact      *string  `json:"teamContact"`
	ProductAreaID    *string  `json:"productAreaID"`
	TeamID           *string  `json:"teamID"`
}

type GroupStats struct {
	Email        string `json:"email"`
	Dataproducts int    `json:"dataproducts"`
}
