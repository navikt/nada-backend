package core

import (
	"context"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.KeywordsService = &keywordsService{}

type keywordsService struct {
	keywordsStorage service.KeywordsStorage
}

func (k *keywordsService) GetKeywordsListSortedByPopularity(ctx context.Context) (*service.KeywordsList, error) {
	const op errs.Op = "keywordsService.GetKeywordsListSortedByPopularity"

	kw, err := k.keywordsStorage.GetKeywordsListSortedByPopularity(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return kw, nil
}

func (k *keywordsService) UpdateKeywords(ctx context.Context, input service.UpdateKeywordsDto) error {
	const op errs.Op = "keywordsService.UpdateKeywords"

	// FIXME: make this configurable
	err := ensureUserInGroup(ctx, "nada@nav.no")
	if err != nil {
		return errs.E(op, err)
	}

	err = k.keywordsStorage.UpdateKeywords(ctx, input)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func NewKeywordsService(storage service.KeywordsStorage) *keywordsService {
	return &keywordsService{keywordsStorage: storage}
}
