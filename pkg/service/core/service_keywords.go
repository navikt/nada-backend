package core

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.KeywordsService = &keywordsService{}

type keywordsService struct {
	keywordsStorage service.KeywordsStorage
}

func (k *keywordsService) GetKeywordsListSortedByPopularity(ctx context.Context) (*service.KeywordsList, error) {
	return k.keywordsStorage.GetKeywordsListSortedByPopularity(ctx)
}

func (k *keywordsService) UpdateKeywords(ctx context.Context, input service.UpdateKeywordsDto) error {
	err := ensureUserInGroup(ctx, "nada@nav.no")
	if err != nil {
		return err
	}

	return k.keywordsStorage.UpdateKeywords(ctx, input)
}

func NewKeywordsService(storage service.KeywordsStorage) *keywordsService {
	return &keywordsService{keywordsStorage: storage}
}
