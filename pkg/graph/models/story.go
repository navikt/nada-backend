package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DBStoryView struct {
	ID   uuid.UUID       `json:"id"`
	Type string          `json:"type"`
	Spec json.RawMessage `json:"spec"`
}

type DBStory struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	Owner        Owner         `json:"owner"`
	Description  string        `json:"description"`
	Keywords     []string      `json:"keywords"`
	Created      time.Time     `json:"created"`
	LastModified time.Time     `json:"lastModified"`
	Views        []DBStoryView `json:"views"`
	Draft        bool
}

type GraphStory struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Owner        Owner            `json:"owner"`
	Description  string           `json:"description"`
	Keywords     []string         `json:"keywords"`
	Created      time.Time        `json:"created"`
	LastModified *time.Time       `json:"lastModified"`
	Views        []GraphStoryView `json:"views"`
	Draft        bool
}

type NewStory struct {
	ID               uuid.UUID  `json:"id"`
	Target           *uuid.UUID `json:"target"`
	Group            string     `json:"group"`
	Keywords         []string   `json:"keywords"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	ProductAreaID    *string    `json:"productAreaID"`
	TeamID           *string    `json:"teamID"`
}

func (GraphStory) IsSearchResult() {}

type GraphStoryView interface {
	IsStoryView()
}

type StoryViewHeader struct {
	ID      uuid.UUID `json:"id"`
	Content string    `json:"content"`
	Level   int       `json:"level"`
}

func (StoryViewHeader) IsStoryView() {}

type StoryViewMarkdown struct {
	ID      uuid.UUID `json:"id"`
	Content string    `json:"content"`
}

func (StoryViewMarkdown) IsStoryView() {}

type StoryViewPlotly struct {
	ID     uuid.UUID                `json:"id"`
	Data   []map[string]interface{} `json:"data"`
	Layout map[string]interface{}   `json:"layout"`
	Frames []map[string]interface{} `json:"frames"`
}

func (StoryViewPlotly) IsStoryView() {}

type StoryViewVega struct {
	ID   uuid.UUID              `json:"id"`
	Spec map[string]interface{} `json:"spec"`
}

func (StoryViewVega) IsStoryView() {}

type StoryToken struct {
	ID      uuid.UUID `json:"id"`
	StoryID uuid.UUID `json:"story_id"`
	Token   string    `json:"token"`
}
