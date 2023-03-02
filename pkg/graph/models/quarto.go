package models

import (
	"time"

	"github.com/google/uuid"
)

type Quarto struct {
	// id of the quarto.
	ID uuid.UUID `json:"id"`
	// owner of the quarto.
	Owner *Owner `json:"team"`
	// created is the timestamp for when the quarto was created.
	Created time.Time `json:"created"`
	// lastModified is the timestamp for when the quarto was last modified.
	LastModified time.Time `json:"lastModified"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
	// content is the content of the quarto.
	Content string `json:"content"`
}

// QuartoStory contains the metadata and content of data stories.
type QuartoStory struct {
	// id of the data story.
	ID uuid.UUID `json:"id"`
	// name of the data story.
	Name string `json:"name"`
	// filename of the quarto story.
	Filename string `json:"filename"`
	// url for the story in bucket.
	URL string `json:"url"`
}

// NewQuartoStory contains the metadata and content of quarto stories.
type NewQuartoStory struct {
	// name of the quarto story.
	Name string `json:"name"`
	// description of the quarto story.
	Description string `json:"description"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
}
