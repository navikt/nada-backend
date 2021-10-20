package models

import "github.com/google/uuid"

type CollectionElement interface {
	IsCollectionElement()
}
type Collection struct {
	ID       uuid.UUID           `json:"id"`
	Elements []CollectionElement `json:"elements"`
}
