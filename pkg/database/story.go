package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetStories(ctx context.Context) ([]*models.DBStory, error) {
	stories, err := r.querier.GetStories(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.DBStory, len(stories))
	for i, s := range stories {
		ret[i] = storyFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) GetStoryView(ctx context.Context, id uuid.UUID) (*models.DBStoryView, error) {
	storyView, err := r.querier.GetStoryView(ctx, id)
	if err != nil {
		return nil, err
	}

	return storyViewFromSQL(storyView), nil
}

func (r *Repo) GetStoryViews(ctx context.Context, storyID uuid.UUID) ([]*models.DBStoryView, error) {
	storyViews, err := r.querier.GetStoryViews(ctx, storyID)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.DBStoryView, len(storyViews))
	for i, s := range storyViews {
		ret[i] = storyViewFromSQL(s)
	}

	return ret, nil
}

func (r *Repo) PublishStory(ctx context.Context, draftID uuid.UUID, group string, keywords []string) (*models.DBStory, error) {
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
		Name:        draft.Name,
		Grp:         group,
		Description: sql.NullString{String: "", Valid: false},
		Keywords:    keywords,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when create story failed")
		}
		return nil, err
	}

	for _, view := range draftViews {
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return storyFromSQL(story), nil
}

func (r *Repo) UpdateStory(ctx context.Context, draftID, target uuid.UUID) (*models.DBStory, error) {
	draftViews, err := r.querier.GetStoryViewDrafts(ctx, draftID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.querier.WithTx(tx)
	if err := querier.DeleteStoryViews(ctx, target); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when delete story views failed")
		}
		return nil, err
	}

	for _, view := range draftViews {
		_, err := querier.CreateStoryView(ctx, gensql.CreateStoryViewParams{
			StoryID: target,
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

	if err := querier.UpdateLastModifiedOnStory(ctx, target); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when create story view failed")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	story, err := r.querier.GetStory(ctx, target)
	if err != nil {
		return nil, err
	}

	return storyFromSQL(story), nil
}

func (r *Repo) UpdateStoryMetadata(ctx context.Context, id uuid.UUID, name string, keywords []string) (*models.DBStory, error) {
	story, err := r.querier.GetStory(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := r.querier.UpdateStory(ctx, gensql.UpdateStoryParams{
		Name:        name,
		Grp:         story.Group,
		Description: sql.NullString{String: "", Valid: false},
		Keywords:    keywords,
		ID:          id,
	})
	if err != nil {
		return nil, err
	}

	return storyFromSQL(updated), nil
}

func (r *Repo) GetStoryToken(ctx context.Context, storyID uuid.UUID) (*models.StoryToken, error) {
	token, err := r.querier.GetStoryToken(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return &models.StoryToken{
		ID:    storyID,
		Token: token.Token.String(),
	}, nil
}

func (r *Repo) GetStoriesByGroups(ctx context.Context, groups []string) ([]*models.DBStory, error) {
	stories, err := r.querier.GetStoriesByGroups(ctx, groups)
	if err != nil {
		return nil, err
	}

	dbStories := make([]*models.DBStory, len(stories))
	for i, s := range stories {
		dbStories[i] = storyFromSQL(s)
	}
	return dbStories, nil
}

func (r *Repo) GetStoryFromToken(ctx context.Context, token uuid.UUID) (*models.DBStory, error) {
	story, err := r.querier.GetStoryFromToken(ctx, token)
	if err != nil {
		return nil, err
	}

	views, err := r.querier.GetStoryViews(ctx, story.ID)
	if err != nil {
		return nil, err
	}

	var vs []models.DBStoryView
	for _, v := range views {
		vs = append(vs, models.DBStoryView{
			ID:   v.ID,
			Type: string(v.Type),
			Spec: v.Spec,
		})
	}

	return &models.DBStory{
		ID:           story.ID,
		Name:         story.Name,
		Group:        story.Group,
		Description:  story.Description.String,
		Keywords:     story.Keywords,
		Created:      story.Created,
		LastModified: story.LastModified,
		Views:        vs,
	}, err
}

func (r *Repo) DeleteStory(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	querier := r.querier.WithTx(tx)

	if err := querier.DeleteStory(ctx, id); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when delete story failed")
		}
		return err
	}

	if err := querier.DeleteStoryViews(ctx, id); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("unable to roll back when delete story views failed")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func storyViewFromSQL(s gensql.StoryView) *models.DBStoryView {
	return &models.DBStoryView{
		ID:   s.ID,
		Type: string(s.Type),
		Spec: s.Spec,
	}
}

func storyFromSQL(s gensql.Story) *models.DBStory {
	return &models.DBStory{
		ID:           s.ID,
		Name:         s.Name,
		Group:        s.Group,
		Description:  s.Description.String,
		Keywords:     s.Keywords,
		Created:      s.Created,
		LastModified: s.LastModified,
	}
}
