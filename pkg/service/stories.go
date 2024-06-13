package service

import (
	"cloud.google.com/go/storage"
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type StoryStorage interface {
	GetStoriesWithTeamkatalogenByGroups(ctx context.Context, groups []string) ([]Story, error)
	GetStoriesWithTeamkatalogenByIDs(ctx context.Context, ids []uuid.UUID) ([]Story, error)
	GetStoriesNumberByTeam(ctx context.Context, teamID string) (int64, error)
	GetStoriesByTeamID(ctx context.Context, teamIDs []string) ([]*Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
	CreateStory(ctx context.Context, creator string, newStory *NewStory) (*Story, error)
	DeleteStory(ctx context.Context, id uuid.UUID) error
	UpdateStory(ctx context.Context, id uuid.UUID, input UpdateStoryDto) (*Story, error)
}

type StoryAPI interface {
	WriteFilesToBucket(ctx context.Context, storyID string, files []*UploadFile, cleanupOnFailure bool) error
	WriteFileToBucket(ctx context.Context, gcsPath string, data []byte) error
	DeleteStoryFolder(ctx context.Context, storyID string) error
	GetIndexHtmlPath(ctx context.Context, prefix string) (string, error)
	GetObject(ctx context.Context, path string) (*storage.ObjectAttrs, []byte, error)
	UploadFile(ctx context.Context, name string, file multipart.File) error
	DeleteObjectsWithPrefix(ctx context.Context, prefix string) error
}

type StoryService interface {
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
	CreateStory(ctx context.Context, newStory *NewStory, files []*UploadFile) (*Story, error)
	CreateStoryWithTeamAndProductArea(ctx context.Context, newStory *NewStory) (*Story, error)
	DeleteStory(ctx context.Context, id string) (*Story, error)
	UpdateStory(ctx context.Context, id string, input UpdateStoryDto) (*Story, error)
	GetObject(ctx context.Context, path string) (*storage.ObjectAttrs, []byte, error)
	RecreateStoryFiles(ctx context.Context, id string, files []*UploadFile) error
	AppendStoryFiles(ctx context.Context, id string, files []*UploadFile) error
	GetIndexHtmlPath(ctx context.Context, prefix string) (string, error)
}

type UploadFile struct {
	// path of the file uploaded
	Path string `json:"path"`
	// file data
	Data []byte `json:"file"`
}

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
