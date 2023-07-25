package database

import (
	"context"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) KeywordsSortedByPopularity(ctx context.Context) ([]*models.Keyword, error) {
	ks, err := r.querier.GetKeywords(ctx)
	if err != nil {
		return nil, err
	}

	km := make([]*models.Keyword, len(ks))
	for i, k := range ks {
		km[i] = &models.Keyword{
			Keyword: k.Keyword,
			Count:   int(k.Count),
		}
	}

	return km, nil
}

func (r *Repo) UpdateKeywords(ctx context.Context, updateInfo models.UpdateKeywords) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	querier := r.querier.WithTx(tx)

	if updateInfo.ObsoleteKeywords != nil {
		for _, kw := range updateInfo.ObsoleteKeywords {
			err := querier.RemoveKeywordInDatasets(ctx, kw)
			if err != nil {
				return err
			}

			err = querier.RemoveKeywordInStories(ctx, kw)
			if err != nil {
				return err
			}
		}
	}

	if updateInfo.ReplacedKeywords != nil {
		for i, kw := range updateInfo.ReplacedKeywords {
			err := querier.ReplaceKeywordInDatasets(ctx, gensql.ReplaceKeywordInDatasetsParams{
				Keyword:           kw,
				NewTextForKeyword: updateInfo.NewText[i],
			})
			if err != nil {
				return err
			}

			err = querier.ReplaceKeywordInStories(ctx, gensql.ReplaceKeywordInStoriesParams{
				Keyword:           kw,
				NewTextForKeyword: updateInfo.NewText[i],
			})
			if err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
