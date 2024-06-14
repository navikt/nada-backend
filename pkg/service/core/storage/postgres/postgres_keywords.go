package postgres

import (
	"context"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.KeywordsStorage = &keywordsStorage{}

type keywordsStorage struct {
	db *database.Repo
}

func (s *keywordsStorage) GetKeywordsListSortedByPopularity(ctx context.Context) (*service.KeywordsList, error) {
	ks, err := s.db.Querier.GetKeywords(ctx)
	if err != nil {
		return nil, err
	}

	km := make([]service.KeywordItem, len(ks))
	for i, k := range ks {
		km[i] = service.KeywordItem{
			Keyword: k.Keyword,
			Count:   int(k.Count),
		}
	}

	return &service.KeywordsList{
		KeywordItems: km,
	}, nil
}

func (s *keywordsStorage) UpdateKeywords(ctx context.Context, input service.UpdateKeywordsDto) error {
	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	querier := s.db.Querier.WithTx(tx)

	if input.ObsoleteKeywords != nil {
		for _, kw := range input.ObsoleteKeywords {
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

	if input.ReplacedKeywords != nil {
		for i, kw := range input.ReplacedKeywords {
			err := querier.ReplaceKeywordInDatasets(ctx, gensql.ReplaceKeywordInDatasetsParams{
				Keyword:           kw,
				NewTextForKeyword: input.NewText[i],
			})
			if err != nil {
				return err
			}
			err = querier.ReplaceKeywordInStories(ctx, gensql.ReplaceKeywordInStoriesParams{
				Keyword:           kw,
				NewTextForKeyword: input.NewText[i],
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

func NewKeywordsStorage(db *database.Repo) *keywordsStorage {
	return &keywordsStorage{db: db}
}
