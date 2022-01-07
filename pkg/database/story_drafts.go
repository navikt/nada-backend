package database

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateStoryDraft(ctx context.Context, story *models.Story) (uuid.UUID, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return uuid.UUID{}, err
	}

	querier := r.querier.WithTx(tx)
	ret, err := querier.CreateStoryDraft(ctx, story.Name)
	if err != nil {
		return uuid.UUID{}, err
	}

	for i, view := range story.Views {
		viewID, err := r.createStoryViewDraft(ctx, querier, ret.ID, view.Type, view.Spec, i)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Errorf("unable to create view %v", i)
			}
			return uuid.UUID{}, err
		}

		if view.Type == models.StoryViewTypePlotly {
			data := map[string]interface{}{"id": viewID}

			_, err := r.createStoryViewDraft(ctx, querier, ret.ID, models.StoryViewTypeGraphURI, data, i)
			if err != nil {
				if err := tx.Rollback(); err != nil {
					r.log.WithError(err).Errorf("unable to create view %v", i)
				}
				return uuid.UUID{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.UUID{}, err
	}
	return ret.ID, nil
}

func (r *Repo) createStoryViewDraft(ctx context.Context, querier *gensql.Queries, storyID uuid.UUID, viewType models.StoryViewType, viewSpec map[string]interface{}, sort int) (uuid.UUID, error) {
	payload, err := json.Marshal(viewSpec)
	if err != nil {
		return uuid.UUID{}, err
	}

	viewDraft, err := querier.CreateStoryViewDraft(ctx, gensql.CreateStoryViewDraftParams{
		StoryID: storyID,
		Type:    gensql.StoryViewType(viewType),
		Spec:    json.RawMessage(payload),
		Sort:    int32(sort),
	})
	return viewDraft.ID, err
}

func (r *Repo) GetStoryDraft(ctx context.Context, id uuid.UUID) (*models.Story, error) {
	story, err := r.querier.GetStoryDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	return storyDraftFromSQL(story), nil
}

func (r *Repo) GetStoryDrafts(ctx context.Context) ([]*models.Story, error) {
	stories, err := r.querier.GetStoryDrafts(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.Story, len(stories))
	for i, s := range stories {
		ret[i] = storyDraftFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStoryViewDraft(ctx context.Context, id uuid.UUID) (*models.StoryView, error) {
	storyView, err := r.querier.GetStoryViewDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	return storyViewDraftFromSQL(storyView), nil
}

func (r *Repo) GetStoryViewDraftsWithoutFigures(ctx context.Context, storyID uuid.UUID) ([]*models.StoryView, error) {
	storyViews, err := r.querier.GetStoryViewDraftsWithoutFigures(ctx, storyID)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.StoryView, len(storyViews))
	for i, s := range storyViews {
		ret[i] = storyViewDraftFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStoryViewDrafts(ctx context.Context, storyID uuid.UUID) ([]*models.StoryView, error) {
	storyViews, err := r.querier.GetStoryViewDrafts(ctx, storyID)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.StoryView, len(storyViews))
	for i, s := range storyViews {
		ret[i] = storyViewDraftFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStory(ctx context.Context, id uuid.UUID) (*models.Story, error) {
	story, err := r.querier.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}

	return storyFromSQL(story), nil
}

func storyDraftFromSQL(s gensql.StoryDraft) *models.Story {
	return &models.Story{
		ID:      s.ID,
		Name:    s.Name,
		Created: s.Created,
		Draft:   true,
	}
}

func storyViewDraftFromSQL(s gensql.StoryViewDraft) *models.StoryView {
	spec := map[string]interface{}{}
	_ = json.Unmarshal(s.Spec, &spec)
	return &models.StoryView{
		Type: models.StoryViewType(s.Type),
		Spec: spec,
	}
}
