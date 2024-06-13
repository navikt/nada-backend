package core

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.StoryService = &storyService{}

type storyService struct {
	storyStorage     service.StoryStorage
	teamKatalogenAPI service.TeamKatalogenAPI
	storyAPI         service.StoryAPI
}

func (s *storyService) GetIndexHtmlPath(ctx context.Context, prefix string) (string, error) {
	return s.storyAPI.GetIndexHtmlPath(ctx, prefix)
}

func (s *storyService) AppendStoryFiles(ctx context.Context, id string, files []*service.UploadFile) error {
	err := s.storyAPI.WriteFilesToBucket(ctx, id, files, false)
	if err != nil {
		return err
	}

	return nil
}

func (s *storyService) RecreateStoryFiles(ctx context.Context, id string, files []*service.UploadFile) error {
	err := s.storyAPI.DeleteObjectsWithPrefix(ctx, id)
	if err != nil {
		return err
	}

	err = s.storyAPI.WriteFilesToBucket(ctx, id, files, false)
	if err != nil {
		return err
	}

	return nil
}

func (s *storyService) CreateStoryWithTeamAndProductArea(ctx context.Context, newStory *service.NewStory) (*service.Story, error) {
	if newStory.TeamID != nil {
		teamCatalogURL := s.teamKatalogenAPI.GetTeamCatalogURL(*newStory.TeamID)
		team, err := s.teamKatalogenAPI.GetTeam(ctx, *newStory.TeamID)
		if err != nil {
			return nil, fmt.Errorf("failed to get team: %w", err)
		}

		newStory.TeamkatalogenURL = &teamCatalogURL
		newStory.ProductAreaID = &team.ProductAreaID
	}

	return s.CreateStory(ctx, newStory, nil)
}

func (s *storyService) GetObject(ctx context.Context, path string) (*storage.ObjectAttrs, []byte, error) {
	return s.storyAPI.GetObject(ctx, path)
}

func (s *storyService) CreateStory(ctx context.Context, newStory *service.NewStory, files []*service.UploadFile) (*service.Story, error) {
	creator := auth.GetUser(ctx).Email
	story, err := s.storyStorage.CreateStory(ctx, creator, newStory)

	if err != nil {
		return nil, fmt.Errorf("failed to create story: %w", err)
	}

	err = s.storyAPI.WriteFilesToBucket(ctx, story.ID.String(), files, true)
	if err != nil {
		return nil, fmt.Errorf("failed to write files to bucket: %w", err)
	}

	// FIXME: this extra getstory might not be necessary
	return s.storyStorage.GetStory(ctx, story.ID)
}

func (s *storyService) DeleteStory(ctx context.Context, id string) (*service.Story, error) {
	storyID := uuid.MustParse(id)

	story, apiErr := s.storyStorage.GetStory(ctx, storyID)
	if apiErr != nil {
		return nil, apiErr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(story.Group) {
		return nil, fmt.Errorf("unauthorized")
	}

	err := s.storyStorage.DeleteStory(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete story: %w", err)
	}

	if err := s.storyAPI.DeleteStoryFolder(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete story files: %w", err)
	}

	return story, nil
}

func (s *storyService) UpdateStory(ctx context.Context, id string, input service.UpdateStoryDto) (*service.Story, error) {
	storyUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	existing, apierr := s.storyStorage.GetStory(ctx, storyUUID)
	if apierr != nil {
		return nil, apierr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, fmt.Errorf("user not in the group of the data story")
	}

	story, err := s.storyStorage.UpdateStory(ctx, storyUUID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update story: %w", err)
	}

	return story, nil
}

func (s *storyService) GetStory(ctx context.Context, id uuid.UUID) (*service.Story, error) {
	story, err := s.storyStorage.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}

	return story, nil
}

func NewStoryService(storage service.StoryStorage) *storyService {
	return &storyService{
		storyStorage: storage,
	}
}
