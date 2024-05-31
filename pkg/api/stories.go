package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
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

type UpdateStoryDto struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Keywords         []string `json:"keywords"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	ProductAreaID    *string  `json:"productAreaID"`
	TeamID           *string  `json:"teamID"`
	Group            string   `json:"group"`
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

func parseFiles(ctx context.Context, r *http.Request) ([]*UploadFile, *APIError) {
	err := r.ParseMultipartForm(50 << 20) // Limit your max input length!
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "Error parsing form")
	}

	files := make([]*UploadFile, 0)
	for i := 0; ; i++ {
		pathKey := fmt.Sprintf("files[%d][path]", i)
		fileKey := fmt.Sprintf("files[%d][file]", i)
		path := r.FormValue(pathKey)
		if path == "" {
			break
		}

		file, _, err := r.FormFile(fileKey)
		if err != nil {
			return nil, NewAPIError(http.StatusBadRequest, err, "Error retrieving file")
		}
		defer file.Close()

		files = append(files, &UploadFile{
			Path: path,
			File: file,
		})
	}
	return files, nil
}

func createStory(ctx context.Context, newStory *NewStory, files []*UploadFile) (*Story, *APIError) {
	var storySQL gensql.Story
	var err error
	creator := auth.GetUser(ctx).Email
	if newStory.ID == nil {
		storySQL, err = queries.CreateStory(ctx, gensql.CreateStoryParams{
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           ptrToNullString(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	} else {
		storySQL, err = queries.CreateStoryWithID(ctx, gensql.CreateStoryWithIDParams{
			ID:               *newStory.ID,
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           ptrToNullString(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	}
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to create story")
	}

	if err = WriteFilesToBucket(ctx, storySQL.ID.String(), files); err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to write files to bucket")
	}

	return getStoryMetadata(ctx, storySQL.ID.String())
}

func deleteStory(ctx context.Context, id string) (*Story, *APIError) {
	storyID := uuid.MustParse(id)
	story, apiErr := getStoryMetadata(ctx, id)
	if apiErr != nil {
		return nil, apiErr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(story.Group) {
		return nil, NewAPIError(http.StatusUnauthorized, nil, "Unauthorized")
	}

	err := queries.DeleteStory(ctx, storyID)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to delete story")
	}

	if err := deleteStoryFolder(ctx, id); err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to delete story files")
	}

	return story, nil
}

func updateStory(ctx context.Context, id string, input UpdateStoryDto) (*Story, *APIError) {
	storyUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "Invalid UUID")
	}

	existing, apierr := getStoryMetadata(ctx, id)
	if apierr != nil {
		return nil, apierr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, NewAPIError(http.StatusUnauthorized, fmt.Errorf("unauthorized"), "user not in the group of the data story")
	}

	dbStory, err := queries.UpdateStory(ctx, gensql.UpdateStoryParams{
		ID:               storyUUID,
		Name:             input.Name,
		Description:      input.Description,
		Keywords:         input.Keywords,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		OwnerGroup:       input.Group,
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to update data story")
	}

	return getStoryMetadata(ctx, dbStory.ID.String())
}
