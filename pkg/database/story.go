package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateStory(ctx context.Context, creator string,
	newStory models.NewStory,
) (*models.Story, error) {
	var storySQL gensql.Story
	var err error
	if newStory.ID == nil {
		storySQL, err = r.Querier.CreateStory(ctx, gensql.CreateStoryParams{
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           ptrToNullString(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	} else {
		storySQL, err = r.Querier.CreateStoryWithID(ctx, gensql.CreateStoryWithIDParams{
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
		return nil, err
	}

	return storySQLToGraphql(&storySQL), err
}

func (r *Repo) GetStory(ctx context.Context, id uuid.UUID) (*models.Story, error) {
	storySQL, err := r.Querier.GetStory(ctx, id)

	return storySQLToGraphql(&storySQL), err
}

func (r *Repo) GetStories(ctx context.Context) ([]*models.Story, error) {
	storySQLs, err := r.Querier.GetStories(ctx)
	if err != nil {
		return nil, err
	}

	storyGraphqls := make([]*models.Story, len(storySQLs))
	for idx, story := range storySQLs {
		storyGraphqls[idx] = storySQLToGraphql(&story)
	}

	return storyGraphqls, err
}

func (r *Repo) GetStoriesByProductArea(ctx context.Context, teamsInPA []string) ([]*models.Story, error) {
	stories, err := r.Querier.GetStoriesByProductArea(ctx, teamsInPA)
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.Story, len(stories))
	for idx, s := range stories {
		dbStories[idx] = storySQLToGraphql(&s)
	}

	return dbStories, nil
}

func (r *Repo) GetStoriesByTeam(ctx context.Context, teamID string) ([]*models.Story, error) {
	stories, err := r.Querier.GetStoriesByTeam(ctx, sql.NullString{String: teamID, Valid: true})
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.Story, len(stories))
	for idx, s := range stories {
		dbStories[idx] = storySQLToGraphql(&s)
	}

	return dbStories, nil
}

func (r *Repo) GetStoriesByGroups(ctx context.Context, groups []string) ([]*models.Story, error) {
	dbStories, err := r.Querier.GetStoriesByGroups(ctx, groups)
	if err != nil {
		return nil, err
	}

	stories := make([]*models.Story, len(dbStories))
	for idx, s := range dbStories {
		stories[idx] = storySQLToGraphql(&s)
	}

	return stories, nil
}

func (r *Repo) UpdateStoryMetadata(ctx context.Context, id uuid.UUID, name string, description string, keywords []string, teamkatalogenURL *string, productAreaID *string, teamID *string, group string) (
	*models.Story, error,
) {
	dbStory, err := r.Querier.UpdateStory(ctx, gensql.UpdateStoryParams{
		ID:               id,
		Name:             name,
		Description:      description,
		Keywords:         keywords,
		TeamkatalogenUrl: ptrToNullString(teamkatalogenURL),
		TeamID:           ptrToNullString(teamID),
		OwnerGroup:       group,
	})
	if err != nil {
		return nil, err
	}

	return storySQLToGraphql(&dbStory), nil
}

func (r *Repo) DeleteStory(ctx context.Context, id uuid.UUID) error {
	return r.Querier.DeleteStory(ctx, id)
}

func storySQLToGraphql(story *gensql.Story) *models.Story {
	return &models.Story{
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
	}
}
