package database

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetStories(ctx context.Context) ([]*models.Story, error) {
	stories, err := r.querier.GetStories(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.Story, len(stories))
	for i, s := range stories {
		ret[i] = storyFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStoryView(ctx context.Context, id uuid.UUID) (*models.StoryView, error) {
	storyView, err := r.querier.GetStoryView(ctx, id)
	if err != nil {
		return nil, err
	}

	return storyViewDraftFromSQL(storyView), nil
}

func (r *Repo) GetStoryViewsWithoutFigures(ctx context.Context, storyID uuid.UUID) ([]*models.StoryView, error) {
	storyViews, err := r.querier.GetStoryViewsWithoutFigures(ctx, storyID)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.StoryView, len(storyViews))
	for i, s := range storyViews {
		ret[i] = storyViewFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStoryViews(ctx context.Context, storyID uuid.UUID) ([]*models.StoryView, error) {
	storyViews, err := r.querier.GetStoryViews(ctx, storyID)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.StoryView, len(storyViews))
	for i, s := range storyViews {
		ret[i] = storyViewFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) PublishStory(ctx context.Context, draftID uuid.UUID, group string) (*models.Story, error) {
	draft, err := r.querier.GetStoryDraft(ctx, draftID)
	if err != nil {
		return nil, err
	}

	draftViews, err := r.querier.GetStoryViewDrafts(ctx, draftID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.querier.WithTx(tx)
	story, err := querier.CreateStory(ctx, gensql.CreateStoryParams{
		Name: draft.Name,
		Grp:  group,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when create story failed")
		}
		return nil, err
	}

	for _, view := range draftViews {
		if view.Type != gensql.StoryViewType(models.StoryViewTypeGraphURI) {
			_, err := querier.CreateStoryView(ctx, gensql.CreateStoryViewParams{
				StoryID: story.ID,
				Sort:    view.Sort,
				Type:    view.Type,
				Spec:    view.Spec,
			})
			if err != nil {
				if err := tx.Rollback(); err != nil {
					r.log.WithError(err).Error("unable to roll back when create story view failed")
				}
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return storyFromSQL(story), nil
}

func (r *Repo) UpdateStory(ctx context.Context, draftID, target uuid.UUID, group string) (*models.Story, error) {
	draft, err := r.querier.GetStoryDraft(ctx, draftID)
	if err != nil {
		return nil, err
	}

	draftViews, err := r.querier.GetStoryViewDrafts(ctx, draftID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.querier.WithTx(tx)
	story, err := querier.UpdateStory(ctx, gensql.UpdateStoryParams{
		ID:   target,
		Name: draft.Name,
		Grp:  group,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when create story failed")
		}
		return nil, err
	}

	if err := querier.DeleteStoryViews(ctx, story.ID); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when delete story views failed")
		}
		return nil, err
	}

	for _, view := range draftViews {
		if view.Type != gensql.StoryViewType(models.StoryViewTypeGraphURI) {
			_, err := querier.CreateStoryView(ctx, gensql.CreateStoryViewParams{
				StoryID: story.ID,
				Sort:    view.Sort,
				Type:    view.Type,
				Spec:    view.Spec,
			})
			if err != nil {
				if err := tx.Rollback(); err != nil {
					r.log.WithError(err).Error("unable to roll back when create story view failed")
				}
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return storyFromSQL(story), nil
}

func storyViewFromSQL(s gensql.StoryView) *models.StoryView {
	spec := map[string]interface{}{}
	_ = json.Unmarshal(s.Spec, &spec)
	return &models.StoryView{
		Type: models.StoryViewType(s.Type),
		Spec: spec,
	}
}

func storyFromSQL(s gensql.Story) *models.Story {
	return &models.Story{
		ID:      s.ID,
		Name:    s.Name,
		Group:   s.Group,
		Created: s.Created,
	}
}
