package api

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type KeywordsList struct {
	KeywordItems []KeywordItem `json:"keywordItems"`
}

type KeywordItem struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type UpdateKeywordsDto struct {
	ObsoleteKeywords []string `json:"obsoleteKeywords"`
	ReplacedKeywords []string `json:"replacedKeywords"`
	NewText          []string `json:"newText"`
}

func getKeywordsListSortedByPopularity(ctx context.Context) (*KeywordsList, *APIError) {
	ks, err := queries.GetKeywords(ctx)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to get keywords")
	}

	km := make([]KeywordItem, len(ks))
	for i, k := range ks {
		km[i] = KeywordItem{
			Keyword: k.Keyword,
			Count:   int(k.Count),
		}
	}

	return &KeywordsList{
		KeywordItems: km,
	}, nil
}

// UpdateKeywords is the resolver for the updateKeywords field.
func UpdateKeywords(ctx context.Context, input UpdateKeywordsDto) *APIError {
	err := ensureUserInGroup(ctx, "nada@nav.no")
	if err != nil {
		return NewAPIError(http.StatusForbidden, err, "Failed to ensure user in group")
	}

	tx, err := sqldb.Begin()
	if err != nil {
		return DBErrorToAPIError(err, "Failed to start transaction")
	}
	defer tx.Rollback()

	querier := queries.WithTx(tx)

	if input.ObsoleteKeywords != nil {
		for _, kw := range input.ObsoleteKeywords {
			err := querier.RemoveKeywordInDatasets(ctx, kw)
			if err != nil {
				return DBErrorToAPIError(err, "Failed to remove keyword in datasets")
			}
			err = querier.RemoveKeywordInStories(ctx, kw)
			if err != nil {
				return DBErrorToAPIError(err, "Failed to remove keyword in datasets")
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
				return DBErrorToAPIError(err, "Failed to replace keyword in datasets")
			}
			err = querier.ReplaceKeywordInStories(ctx, gensql.ReplaceKeywordInStoriesParams{
				Keyword:           kw,
				NewTextForKeyword: input.NewText[i],
			})
			if err != nil {
				return DBErrorToAPIError(err, "Failed to replace keyword in datasets")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return DBErrorToAPIError(err, "Failed to commit transaction")
	}

	return nil
}
