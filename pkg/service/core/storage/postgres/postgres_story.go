package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.StoryStorage = &storyStorage{}

type storyStorage struct {
	db *database.Repo
}

func (s *storyStorage) GetStoriesByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesByTeamID"

	sqlStories, err := s.db.Querier.GetStoriesByProductArea(ctx, uuidListToStringList(teamIDs))
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	stories := make([]*service.Story, len(sqlStories))
	for idx, s := range sqlStories {
		stories[idx] = storyFromSQL(&s)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	const op errs.Op = "storyStorage.GetStoriesNumberByTeam"

	n, err := s.db.Querier.GetStoriesNumberByTeam(ctx, uuidToNullString(teamID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, errs.E(errs.Database, op, err)
	}

	return n, nil
}

func (s *storyStorage) UpdateStory(ctx context.Context, id uuid.UUID, input service.UpdateStoryDto) (*service.Story, error) {
	const op errs.Op = "storyStorage.UpdateStory"

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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	st, err := s.GetStory(ctx, dbStory.ID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return st, nil
}

func (s *storyStorage) DeleteStory(ctx context.Context, id uuid.UUID) error {
	const op errs.Op = "storyStorage.DeleteStory"

	err := s.db.Querier.DeleteStory(ctx, id)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *storyStorage) CreateStory(ctx context.Context, creator string, newStory *service.NewStory) (*service.Story, error) {
	const op errs.Op = "storyStorage.CreateStory"

	var storySQL gensql.Story
	var err error

	if newStory.ID == nil {
		storySQL, err = s.db.Querier.CreateStory(ctx, gensql.CreateStoryParams{
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           uuidPtrToNullString(newStory.TeamID),
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
			TeamID:           uuidPtrToNullString(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	}
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	st, err := s.GetStory(ctx, storySQL.ID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return st, nil
}

func (s *storyStorage) GetStory(ctx context.Context, id uuid.UUID) (*service.Story, error) {
	const op errs.Op = "storyStorage.GetStory"

	stories, err := s.GetStoriesWithTeamkatalogenByIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, errs.E(op, err)
	}

	if len(stories) == 0 {
		return nil, errs.E(errs.NotExist, op, fmt.Errorf("story with id %s not found", id))
	}

	return &stories[0], nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByIDs(ctx context.Context, ids []uuid.UUID) ([]service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesWithTeamkatalogenByIDs"

	dbStories, err := s.db.Querier.GetStoriesWithTeamkatalogenByIDs(ctx, ids)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	stories := make([]service.Story, len(dbStories))
	for i, story := range dbStories {
		stories[i] = *storyFromSQL(&story)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByGroups(ctx context.Context, groups []string) ([]service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesWithTeamkatalogenByGroups"

	dbStories, err := s.db.Querier.GetStoriesWithTeamkatalogenByGroups(ctx, groups)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
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
		TeamID:           nullStringToUUIDPtr(story.TeamID),
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
