package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateQuartoStory(ctx context.Context, creator string,
	newQuartoStory models.NewQuartoStory,
) (models.QuartoStory, error) {
	quartoSQL, err := r.querier.CreateQuartoStory(ctx, gensql.CreateQuartoStoryParams{
		Name:             newQuartoStory.Name,
		Creator:          creator,
		Description:      newQuartoStory.Description,
		Keywords:         newQuartoStory.Keywords,
		TeamkatalogenUrl: ptrToNullString(newQuartoStory.TeamkatalogenURL),
		ProductAreaID:    ptrToNullString(newQuartoStory.ProductAreaID),
		TeamID:           ptrToNullString(newQuartoStory.TeamID),
		OwnerGroup:       newQuartoStory.Group,
	})

	return *quartoSQLToGraphql(&quartoSQL), err
}

func (r *Repo) GetQuartoStory(ctx context.Context, id uuid.UUID) (*models.QuartoStory, error) {
	quartoSQL, err := r.querier.GetQuartoStory(ctx, id)

	return quartoSQLToGraphql(&quartoSQL), err
}

func (r *Repo) GetQuartoStories(ctx context.Context) ([]*models.QuartoStory, error) {
	quartoSQLs, err := r.querier.GetQuartoStories(ctx)
	if err != nil {
		return nil, err
	}

	quartoGraphqls := make([]*models.QuartoStory, len(quartoSQLs))
	for idx, quarto := range quartoSQLs {
		quartoGraphqls[idx] = quartoSQLToGraphql(&quarto)
	}

	return quartoGraphqls, err
}

func (r *Repo) GetQuartoStoriesByProductArea(ctx context.Context, paID string) ([]*models.QuartoStory, error) {
	stories, err := r.querier.GetQuartoStoriesByProductArea(
		ctx, sql.NullString{String: paID, Valid: true})
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.QuartoStory, len(stories))
	for idx, s := range stories {
		dbStories[idx] = quartoSQLToGraphql(&s)
	}

	return dbStories, nil
}

func (r *Repo) GetQuartoStoriesByTeam(ctx context.Context, teamID string) ([]*models.QuartoStory, error) {
	stories, err := r.querier.GetQuartoStoriesByTeam(ctx, sql.NullString{String: teamID, Valid: true})
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.QuartoStory, len(stories))
	for idx, s := range stories {
		dbStories[idx] = quartoSQLToGraphql(&s)
	}

	return dbStories, nil
}

func (r *Repo) GetQuartoStoriesByGroups(ctx context.Context, groups []string) ([]*models.QuartoStory, error) {
	stories, err := r.querier.GetQuartoStoriesByGroups(ctx, groups)
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.QuartoStory, len(stories))
	for idx, s := range stories {
		dbStories[idx] = quartoSQLToGraphql(&s)
	}

	return dbStories, nil
}

func (r *Repo) UpdateQuartoStoryMetadata(ctx context.Context, id uuid.UUID, name string, description string, keywords []string, teamkatalogenURL *string, productAreaID *string, teamID *string, group string) (
	*models.QuartoStory, error) {
	dbStory, err := r.querier.UpdateQuartoStory(ctx, gensql.UpdateQuartoStoryParams{
		ID:               id,
		Name:             name,
		Description:      description,
		Keywords:         keywords,
		TeamkatalogenUrl: ptrToNullString(teamkatalogenURL),
		ProductAreaID:    ptrToNullString(productAreaID),
		TeamID:           ptrToNullString(teamID),
		OwnerGroup:       group,
	})
	if err != nil {
		return nil, err
	}

	return quartoSQLToGraphql(&dbStory), nil
}

func (r *Repo) DeleteQuartoStory(ctx context.Context, id uuid.UUID) error {
	return r.querier.DeleteStory(ctx, id)
}

func quartoSQLToGraphql(quarto *gensql.QuartoStory) *models.QuartoStory {
	return &models.QuartoStory{
		ID:            quarto.ID,
		Name:          quarto.Name,
		Creator:       quarto.Creator,
		Created:       quarto.Created,
		LastModified:  &quarto.LastModified,
		Keywords:      quarto.Keywords,
		ProductAreaID: nullStringToPtr(quarto.ProductAreaID),
		TeamID:        nullStringToPtr(quarto.TeamID),
		TeamkatalogenURL: &quarto.TeamkatalogenUrl.String,
		Description:   quarto.Description,
		Group:         quarto.Group,
	}
}
