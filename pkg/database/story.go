package database

import (
	"context"

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
		_, err := querier.CreateStoryViewDraft(ctx, gensql.CreateStoryViewDraftParams{
			StoryID: ret.ID,
			Type:    gensql.StoryViewType(view.Type),
			Spec:    view.Spec,
			Sort:    int32(i),
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Errorf("unable to create view %v", i)
			}
			return uuid.UUID{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.UUID{}, err
	}
	return ret.ID, nil
}
