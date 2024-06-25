package service

import (
	"context"
)

type KeywordsStorage interface {
	GetKeywordsListSortedByPopularity(ctx context.Context) (*KeywordsList, error)
	UpdateKeywords(ctx context.Context, input UpdateKeywordsDto) error
}

type KeywordsService interface {
	GetKeywordsListSortedByPopularity(ctx context.Context) (*KeywordsList, error)
	UpdateKeywords(ctx context.Context, user *User, input UpdateKeywordsDto) error
}

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
