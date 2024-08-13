package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.StoryService = &storyService{}

type storyService struct {
	storyStorage            service.StoryStorage
	teamKatalogenAPI        service.TeamKatalogenAPI
	storyAPI                service.StoryAPI
	createIgnoreMissingTeam bool
}

func (s *storyService) GetIndexHtmlPath(ctx context.Context, prefix string) (string, error) {
	const op = "storyService.GetIndexHtmlPath"

	index, err := s.storyAPI.GetIndexHtmlPath(ctx, prefix)
	if err != nil {
		return "", errs.E(op, err)
	}

	return index, nil
}

func (s *storyService) AppendStoryFiles(ctx context.Context, id uuid.UUID, creatorEmail string, files []*service.UploadFile) error {
	const op = "storyService.AppendStoryFiles"

	story, err := s.storyStorage.GetStory(ctx, id)
	if err != nil {
		return errs.E(op, err)
	}

	if auth.TrimNaisTeamPrefix(story.Group) != creatorEmail {
		return errs.E(errs.Unauthorized, op, errs.UserName(creatorEmail), fmt.Errorf("user %s not in the group of the data story: %s", creatorEmail, story.Group))
	}

	err = s.storyAPI.WriteFilesToBucket(ctx, id.String(), files, false)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *storyService) RecreateStoryFiles(ctx context.Context, id uuid.UUID, creatorEmail string, files []*service.UploadFile) error {
	const op = "storyService.RecreateStoryFiles"

	story, err := s.storyStorage.GetStory(ctx, id)
	if err != nil {
		return errs.E(op, err)
	}

	if auth.TrimNaisTeamPrefix(story.Group) != creatorEmail {
		return errs.E(errs.Unauthorized, op, errs.UserName(creatorEmail), fmt.Errorf("user %s not in the group of the data story: %s", creatorEmail, story.Group))
	}

	_, err = s.storyAPI.DeleteObjectsWithPrefix(ctx, id.String())
	if err != nil {
		return errs.E(op, err)
	}

	err = s.storyAPI.WriteFilesToBucket(ctx, id.String(), files, false)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *storyService) CreateStoryWithTeamAndProductArea(ctx context.Context, creatorEmail string, newStory *service.NewStory) (*service.Story, error) {
	const op = "storyService.CreateStoryWithTeamAndProductArea"

	if newStory.TeamID != nil {
		teamCatalogURL := s.teamKatalogenAPI.GetTeamCatalogURL(*newStory.TeamID)
		team, err := s.teamKatalogenAPI.GetTeam(ctx, *newStory.TeamID)
		if err != nil {
			if !s.createIgnoreMissingTeam {
				return nil, errs.E(op, err)
			}
		}

		newStory.TeamkatalogenURL = &teamCatalogURL
		if team != nil {
			newStory.ProductAreaID = &team.ProductAreaID
		}
	}

	story, err := s.CreateStory(ctx, creatorEmail, newStory, nil)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (s *storyService) GetObject(ctx context.Context, path string) (*service.ObjectWithData, error) {
	const op = "storyService.GetObject"

	obj, err := s.storyAPI.GetObject(ctx, path)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return obj, nil
}

func (s *storyService) CreateStory(ctx context.Context, creatorEmail string, newStory *service.NewStory, files []*service.UploadFile) (*service.Story, error) {
	const op = "storyService.CreateStory"

	story, err := s.storyStorage.CreateStory(ctx, creatorEmail, newStory)
	if err != nil {
		return nil, errs.E(op, err)
	}

	err = s.storyAPI.WriteFilesToBucket(ctx, story.ID.String(), files, true)
	if err != nil {
		return nil, errs.E(op, err)
	}

	st, err := s.storyStorage.GetStory(ctx, story.ID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return st, nil
}

func (s *storyService) DeleteStory(ctx context.Context, user *service.User, storyID uuid.UUID) (*service.Story, error) {
	const op = "storyService.DeleteStory"

	story, err := s.storyStorage.GetStory(ctx, storyID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if !user.GoogleGroups.Contains(story.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not in the group of the data story: %s", story.Group))
	}

	err = s.storyStorage.DeleteStory(ctx, storyID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if err := s.storyAPI.DeleteStoryFolder(ctx, storyID.String()); err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (s *storyService) UpdateStory(ctx context.Context, user *service.User, storyID uuid.UUID, input service.UpdateStoryDto) (*service.Story, error) {
	const op = "storyService.UpdateStory"

	existing, err := s.storyStorage.GetStory(ctx, storyID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user not in the group of the data story: %s", existing.Group))
	}

	story, err := s.storyStorage.UpdateStory(ctx, storyID, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (s *storyService) GetStory(ctx context.Context, storyID uuid.UUID) (*service.Story, error) {
	const op = "storyService.GetStory"

	story, err := s.storyStorage.GetStory(ctx, storyID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func NewStoryService(
	storyStorage service.StoryStorage,
	teamKatalogenAPI service.TeamKatalogenAPI,
	storyAPI service.StoryAPI,
	createIgnoreMissingTeam bool,
) *storyService {
	return &storyService{
		storyStorage:            storyStorage,
		teamKatalogenAPI:        teamKatalogenAPI,
		storyAPI:                storyAPI,
		createIgnoreMissingTeam: createIgnoreMissingTeam,
	}
}
