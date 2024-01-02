package models

import (
	"time"

	"github.com/google/uuid"
)

// Story contains the metadata and content of data stories.
type Story struct {
	// id of the data story.
	ID uuid.UUID `json:"id"`
	// name of the data story.
	Name string `json:"name"`
	// creator of the data story.
	Creator string `json:"creator"`
	// description of the data story.
	Description string `json:"description"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
	// teamkatalogenURL of the creator.
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	// Id of the creator's team.
	TeamID *string `json:"teamID"`
	// created is the timestamp for when the data story was created.
	Created time.Time `json:"created"`
	// lastModified is the timestamp for when the dataproduct was last modified.
	LastModified *time.Time `json:"lastModified"`
	// group is the owner group of the data story.
	Group string `json:"group"`
}

// NewStory contains the metadata and content of data stories.
type NewStory struct {
	// id of data story.
	ID *uuid.UUID `json:"id"`
	// name of the data story.
	Name string `json:"name"`
	// description of the data story.
	Description *string `json:"description"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
	// teamkatalogenURL of the creator.
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	// Id of the creator's product area.
	ProductAreaID *string `json:"productAreaID"`
	// Id of the creator's team.
	TeamID *string `json:"teamID"`
	// group is the owner group of the data story.
	Group string `json:"group"`
}

func (Story) IsSearchResult() {}
