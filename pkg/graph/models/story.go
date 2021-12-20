package models

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type StoryView struct {
	Type StoryViewType          `json:"type"`
	Spec map[string]interface{} `json:"spec"`
}

type Story struct {
	ID           uuid.UUID   `json:"id"`
	Name         string      `json:"name"`
	Group        string      `json:"group"`
	Created      time.Time   `json:"created"`
	LastModified time.Time   `json:"lastModified"`
	Views        []StoryView `json:"views"`
	Draft        bool
}

type StoryViewType string

const (
	StoryViewTypeMarkdown StoryViewType = "markdown"
	StoryViewTypeHeader   StoryViewType = "header"
	StoryViewTypePlotly   StoryViewType = "plotly"
)

var AllStoryViewType = []StoryViewType{
	StoryViewTypeMarkdown,
	StoryViewTypeHeader,
	StoryViewTypePlotly,
}

func (e StoryViewType) IsValid() bool {
	switch e {
	case StoryViewTypeMarkdown, StoryViewTypeHeader, StoryViewTypePlotly:
		return true
	}
	return false
}

func (e StoryViewType) String() string {
	return string(e)
}

func (e *StoryViewType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = StoryViewType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid StoryViewType", str)
	}
	return nil
}

func (e StoryViewType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
