package core

import (
	"context"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.KeywordsService = &keywordsService{}

type keywordsService struct {
	keywordsStorage service.KeywordsStorage
	adminGroup      string
}

func (k *keywordsService) GetKeywordsListSortedByPopularity(ctx context.Context) (*service.KeywordsList, error) {
	const op errs.Op = "keywordsService.GetKeywordsListSortedByPopularity"

	kw, err := k.keywordsStorage.GetKeywordsListSortedByPopularity(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return kw, nil
}

func (k *keywordsService) UpdateKeywords(ctx context.Context, user *service.User, input service.UpdateKeywordsDto) error {
	const op errs.Op = "keywordsService.UpdateKeywords"

	err := ensureUserInGroup(user, k.adminGroup)
	if err != nil {
		return errs.E(op, err)
	}

	err = k.keywordsStorage.UpdateKeywords(ctx, input)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func NewKeywordsService(storage service.KeywordsStorage, adminGroup string) *keywordsService {
	return &keywordsService{
		keywordsStorage: storage,
		adminGroup:      adminGroup,
	}
}
