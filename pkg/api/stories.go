package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
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
	Group           string  `json:"group"`
	TeamName        *string `json:"teamName"`
	ProductAreaName string  `json:"productAreaName"`
}

func getStoryMetadata(ctx context.Context, id string) (*Story, *APIError) {
	storyID := uuid.MustParse(id)
	storySQLs, err := queries.GetStoriesWithTeamkatalogenByIDs(ctx, []uuid.UUID{storyID})
	if err != nil {
		return nil, &APIError{
			HttpStatus: http.StatusInternalServerError,
			Err:        err,
			Message:    "fetching existing story metadata",
		}
	}

	return storyFromSQL(&storySQLs[0]), nil
}

func storyFromSQL(story *gensql.StoryWithTeamkatalogenView) *Story {
	return &Story{
		ID:               story.ID,
		Name:             story.Name,
		Creator:          story.Creator,
		Created:          story.Created,
		LastModified:     &story.LastModified,
		Keywords:         story.Keywords,
		TeamID:           nullStringToPtr(story.TeamID),
		TeamkatalogenURL: nullStringToPtr(story.TeamkatalogenUrl),
		Description:      story.Description,
		Group:            story.Group,
		TeamName:         nullStringToPtr(story.TeamName),
		ProductAreaName:  nullStringToString(story.PaName),
	}
}
