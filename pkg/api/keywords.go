package api

import (
	"context"
)

type KeywordsList struct {
	KeywordItems []KeywordItem `json:"keywordItems"`
}

type KeywordItem struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

func getKeywordsListSortedByPopularity(ctx context.Context) (*KeywordsList, *APIError) {
	ks, err := querier.GetKeywords(ctx)
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
