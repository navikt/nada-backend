package handlers

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/errs"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
)

type KeywordsHandler struct {
	keywordsService service.KeywordsService
}

func (h *KeywordsHandler) GetKeywordsListSortedByPopularity(ctx context.Context, _ *http.Request, _ any) (*service.KeywordsList, error) {
	const op errs.Op = "KeywordsHandler.GetKeywordsListSortedByPopularity"

	keywords, err := h.keywordsService.GetKeywordsListSortedByPopularity(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return keywords, nil
}

func (h *KeywordsHandler) UpdateKeywords(ctx context.Context, _ *http.Request, input service.UpdateKeywordsDto) (*transport.Empty, error) {
	const op errs.Op = "KeywordsHandler.UpdateKeywords"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err := h.keywordsService.UpdateKeywords(ctx, user, input)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func NewKeywordsHandler(service service.KeywordsService) *KeywordsHandler {
	return &KeywordsHandler{keywordsService: service}
}
