package models

import (
	"time"

	"github.com/google/uuid"
)

type CollectionElement interface {
	IsCollectionElement()
}

type Collection struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Keywords     []string  `json:"keywords"`
	Owner        Owner     `json:"owner"`
	Slug         string    `json:"slug"`
}

type NewCollection struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Group       string   `json:"group"`
	Keywords    []string `json:"keywords"`
}

type UpdateCollection struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Slug        *string  `json:"slug"`
	Keywords    []string `json:"keywords"`
}
