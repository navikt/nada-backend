package models

import (
	"time"

	"github.com/google/uuid"
)

type Dataproduct struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Created      time.Time       `json:"created"`
	LastModified time.Time       `json:"last_modified"`
	Description  *string         `json:"description"`
	Slug         string          `json:"slug"`
	Repo         *string         `json:"repo"`
	Pii          bool            `json:"pii"`
	Keywords     []string        `json:"keywords"`
	Owner        *Owner          `json:"owner"`
	Type         DataproductType `json:"type"`
	SomeRelation string
}

func (Dataproduct) IsCollectionElement() {}
