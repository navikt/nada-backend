package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.StoryStorage = &storyStorage{}

type storyStorage struct {
	db *database.Repo
}

func (s *storyStorage) GetStoriesByTeamID(ctx context.Context, teamIDs []string) ([]*service.Story, error) {
	sqlStories, err := s.db.Querier.GetStoriesByProductArea(ctx, teamIDs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*service.Story{}, nil
		}

		return nil, err
	}

	stories := make([]*service.Story, len(sqlStories))
	for idx, s := range sqlStories {
		stories[idx] = storyFromSQL(&s)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesNumberByTeam(ctx context.Context, teamID string) (int64, error) {
	n, err := s.db.Querier.GetStoriesNumberByTeam(ctx, ptrToNullString(&teamID))
	if err != nil {
		return 0, fmt.Errorf("failed to get stories number: %w", err)
	}

	return n, nil
}

func (s *storyStorage) UpdateStory(ctx context.Context, id uuid.UUID, input service.UpdateStoryDto) (*service.Story, error) {
	dbStory, err := s.db.Querier.UpdateStory(ctx, gensql.UpdateStoryParams{
		ID:               id,
		Name:             input.Name,
		Description:      input.Description,
		Keywords:         input.Keywords,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		OwnerGroup:       input.Group,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update data story: %w", err)
	}

	// FIXME: is this really needed? maybe some updated_at timestamp thingie?
	return s.GetStory(ctx, dbStory.ID)
}

func (s *storyStorage) DeleteStory(ctx context.Context, id uuid.UUID) error {
	return s.db.Querier.DeleteStory(ctx, id)
}

func (s *storyStorage) CreateStory(ctx context.Context, creator string, newStory *service.NewStory) (*service.Story, error) {
	var storySQL gensql.Story
	var err error

	if newStory.ID == nil {
		storySQL, err = s.db.Querier.CreateStory(ctx, gensql.CreateStoryParams{
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           ptrToNullString(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	} else {
		storySQL, err = s.db.Querier.CreateStoryWithID(ctx, gensql.CreateStoryWithIDParams{
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
		return nil, fmt.Errorf("failed to create story: %w", err)
	}

	return s.GetStory(ctx, storySQL.ID)
}

func (s *storyStorage) GetStory(ctx context.Context, id uuid.UUID) (*service.Story, error) {
	stories, err := s.GetStoriesWithTeamkatalogenByIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, fmt.Errorf("failed to get story: %w", err)
	}

	return &stories[0], nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByIDs(ctx context.Context, ids []uuid.UUID) ([]service.Story, error) {
	dbStories, err := s.db.Querier.GetStoriesWithTeamkatalogenByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get stories with teamkatalogen by ids: %w", err)
	}

	stories := make([]service.Story, len(dbStories))
	for i, story := range dbStories {
		stories[i] = *storyFromSQL(&story)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByGroups(ctx context.Context, groups []string) ([]service.Story, error) {
	dbStories, err := s.db.Querier.GetStoriesWithTeamkatalogenByGroups(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to get stories with teamkatalogen by groups: %w", err)
	}

	stories := make([]service.Story, len(dbStories))
	for i, story := range dbStories {
		stories[i] = *storyFromSQL(&story)
	}

	return stories, nil
}

func storyFromSQL(story *gensql.StoryWithTeamkatalogenView) *service.Story {
	return &service.Story{
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

func NewStoryStorage(db *database.Repo) *storyStorage {
	return &storyStorage{
		db: db,
	}
}
