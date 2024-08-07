package service

import (
	"context"
	"io"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
)

type StoryStorage interface {
	GetStoriesWithTeamkatalogenByGroups(ctx context.Context, groups []string) ([]*Story, error)
	GetStoriesWithTeamkatalogenByIDs(ctx context.Context, ids []uuid.UUID) ([]*Story, error)
	GetStoriesNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
	GetStoriesByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
	CreateStory(ctx context.Context, creator string, newStory *NewStory) (*Story, error)
	DeleteStory(ctx context.Context, id uuid.UUID) error
	UpdateStory(ctx context.Context, id uuid.UUID, input UpdateStoryDto) (*Story, error)
}

type StoryAPI interface {
	WriteFilesToBucket(ctx context.Context, storyID string, files []*UploadFile, cleanupOnFailure bool) error
	WriteFileToBucket(ctx context.Context, pathPrefix string, file *UploadFile) error
	DeleteStoryFolder(ctx context.Context, storyID string) error
	GetIndexHtmlPath(ctx context.Context, prefix string) (string, error)
	GetObject(ctx context.Context, path string) (*ObjectWithData, error)
	DeleteObjectsWithPrefix(ctx context.Context, prefix string) (int, error)
}

type StoryService interface {
	GetStory(ctx context.Context, id uuid.UUID) (*Story, error)
	CreateStory(ctx context.Context, creatorEmail string, newStory *NewStory, files []*UploadFile) (*Story, error)
	CreateStoryWithTeamAndProductArea(ctx context.Context, creatorEmail string, newStory *NewStory) (*Story, error)
	DeleteStory(ctx context.Context, user *User, id uuid.UUID) (*Story, error)
	UpdateStory(ctx context.Context, user *User, id uuid.UUID, input UpdateStoryDto) (*Story, error)
	GetObject(ctx context.Context, path string) (*ObjectWithData, error)
	RecreateStoryFiles(ctx context.Context, id uuid.UUID, creatorEmail string, files []*UploadFile) error
	AppendStoryFiles(ctx context.Context, id uuid.UUID, creatorEmail string, files []*UploadFile) error
	GetIndexHtmlPath(ctx context.Context, prefix string) (string, error)
}

type UploadFile struct {
	// path of the file uploaded
	Path string `json:"path"`
	// file data
	ReadCloser io.ReadCloser
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
	TeamID *uuid.UUID `json:"teamID"`
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
	ProductAreaID *uuid.UUID `json:"productAreaID"`
	// Id of the creator's team.
	TeamID *uuid.UUID `json:"teamID"`
	// group is the owner group of the data story.
	Group string `json:"group"`
}

func (s NewStory) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Name, validation.Required),
		validation.Field(&s.Group, validation.Required),
	)
}

type UpdateStoryDto struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Keywords         []string   `json:"keywords"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	ProductAreaID    *uuid.UUID `json:"productAreaID"`
	TeamID           *uuid.UUID `json:"teamID"`
	Group            string     `json:"group"`
}

type Object struct {
	Name   string
	Bucket string
	Attrs  Attributes
}

type Attributes struct {
	ContentType     string
	ContentEncoding string
	Size            int64
	SizeStr         string
}

type ObjectWithData struct {
	*Object
	Data []byte
}
