package models

import (
	"time"

	"github.com/google/uuid"
)

type Quarto struct {
	// id of the quarto.
	ID uuid.UUID `json:"id"`
	// name of the quarto.
	Team *Owner `json:"team"`
	// created is the timestamp for when the quarto was created.
	Created time.Time `json:"created"`
	// lastModified is the timestamp for when the quarto was last modified.
	LastModified *time.Time `json:"lastModified"`
	// owner of the quarto. Changes to the quarto can only be done by a member of the owner.
	Owner *Owner `json:"owner"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
	// content is the content of the quarto.
	Content string `json:"content"`
}
