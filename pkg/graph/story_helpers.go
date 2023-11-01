package graph

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

func storyFromDB(story *models.DBStory) (*models.GraphStory, error) {
	views, err := storyViewsFromDB(story.Views)
	if err != nil {
		return nil, err
	}

	return &models.GraphStory{
		ID:           story.ID,
		Name:         story.Name,
		Created:      story.Created,
		LastModified: ptrTime(story.LastModified),
		Description:  story.Description,
		Keywords:     story.Keywords,
		Owner:        story.Owner,
		Views:        views,
	}, nil
}

func storyViewsFromDB(view []models.DBStoryView) ([]models.GraphStoryView, error) {
	ret := make([]models.GraphStoryView, len(view))
	for i, s := range view {
		var err error
		ret[i], err = storyViewFromDB(&s)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func storyViewFromDB(view *models.DBStoryView) (models.GraphStoryView, error) {
	var ret models.GraphStoryView
	switch view.Type {
	case "header":
		sv := models.StoryViewHeader{
			ID: view.ID,
		}
		if err := json.Unmarshal(view.Spec, &sv); err != nil {
			return nil, err
		}
		ret = sv
	case "markdown":
		sv := models.StoryViewMarkdown{
			ID: view.ID,
		}
		if err := json.Unmarshal(view.Spec, &sv); err != nil {
			return nil, err
		}
		ret = sv
	case "plotly":
		sv := models.StoryViewPlotly{
			ID: view.ID,
		}
		if err := json.Unmarshal(view.Spec, &sv); err != nil {
			return nil, err
		}
		ret = sv
	case "vega":
		sv := models.StoryViewVega{
			ID: view.ID,
		}
		if err := json.Unmarshal(view.Spec, &sv.Spec); err != nil {
			return nil, err
		}
		ret = sv
	default:
		return nil, fmt.Errorf("unknown story type %q", view.Type)
	}
	return ret, nil
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
