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
	Repo         *string   `json:"repo"`
	Keywords     []string  `json:"keywords"`
	Owner        *Owner    `json:"owner"`
}

func (Dataproduct) IsSearchResult() {}

type NewDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Repo             *string  `json:"repo"`
	Keywords         []string `json:"keywords"`
	Group            string   `json:"group"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	Requesters       []string `json:"requesters"`
	Metadata         BigqueryMetadata
}

type UpdateDataproduct struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Repo             *string  `json:"repo"`
	Pii              bool     `json:"pii"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	Keywords         []string `json:"keywords"`
	Requesters       []string `json:"requesters"`
}

type Keyword struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type GroupStats struct {
	Email        string `json:"email"`
	Dataproducts int    `json:"dataproducts"`
}
