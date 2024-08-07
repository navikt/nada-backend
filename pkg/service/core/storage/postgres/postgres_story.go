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

type StoryWithTeamkatalogenView gensql.StoryWithTeamkatalogenView

func (s StoryWithTeamkatalogenView) To() (*service.Story, error) {
	return &service.Story{
		ID:               s.ID,
		Name:             s.Name,
		Creator:          s.Creator,
		Created:          s.Created,
		LastModified:     &s.LastModified,
		Keywords:         s.Keywords,
		TeamID:           nullUUIDToUUIDPtr(s.TeamID),
		TeamkatalogenURL: nullStringToPtr(s.TeamkatalogenUrl),
		Description:      s.Description,
		Group:            s.Group,
		TeamName:         nullStringToPtr(s.TeamName),
		ProductAreaName:  nullStringToString(s.PaName),
	}, nil
}

type StoryWithTeamkatalogenViewList []gensql.StoryWithTeamkatalogenView

func (s StoryWithTeamkatalogenViewList) To() ([]*service.Story, error) {
	stories := make([]*service.Story, len(s))

	for i, r := range s {
		st, err := From(StoryWithTeamkatalogenView(r))
		if err != nil {
			return nil, err
		}

		stories[i] = st
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesByTeamID"

	raw, err := s.db.Querier.GetStoriesByProductArea(ctx, teamIDs)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	stories, err := From(StoryWithTeamkatalogenViewList(raw))
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	const op errs.Op = "storyStorage.GetStoriesNumberByTeam"

	n, err := s.db.Querier.GetStoriesNumberByTeam(ctx, uuidToNullUUID(teamID))
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
		TeamID:           uuidPtrToNullUUID(input.TeamID),
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

	var story gensql.Story
	var err error

	if newStory.ID == nil {
		story, err = s.db.Querier.CreateStory(ctx, gensql.CreateStoryParams{
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           uuidPtrToNullUUID(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	} else {
		story, err = s.db.Querier.CreateStoryWithID(ctx, gensql.CreateStoryWithIDParams{
			ID:               *newStory.ID,
			Name:             newStory.Name,
			Creator:          creator,
			Description:      ptrToString(newStory.Description),
			Keywords:         newStory.Keywords,
			TeamkatalogenUrl: ptrToNullString(newStory.TeamkatalogenURL),
			TeamID:           uuidPtrToNullUUID(newStory.TeamID),
			OwnerGroup:       newStory.Group,
		})
	}
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	st, err := s.GetStory(ctx, story.ID)
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

	return stories[0], nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByIDs(ctx context.Context, ids []uuid.UUID) ([]*service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesWithTeamkatalogenByIDs"

	raw, err := s.db.Querier.GetStoriesWithTeamkatalogenByIDs(ctx, ids)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	stories, err := From(StoryWithTeamkatalogenViewList(raw))
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return stories, nil
}

func (s *storyStorage) GetStoriesWithTeamkatalogenByGroups(ctx context.Context, groups []string) ([]*service.Story, error) {
	const op errs.Op = "storyStorage.GetStoriesWithTeamkatalogenByGroups"

	raw, err := s.db.Querier.GetStoriesWithTeamkatalogenByGroups(ctx, groups)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	stories, err := From(StoryWithTeamkatalogenViewList(raw))
	if err != nil {
		return nil, errs.E(errs.Internal, op, err)
	}

	return stories, nil
}

func NewStoryStorage(db *database.Repo) *storyStorage {
	return &storyStorage{
		db: db,
	}
}
