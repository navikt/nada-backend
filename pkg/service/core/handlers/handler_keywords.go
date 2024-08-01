package handlers

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
)

type KeywordsHandler struct {
	keywordsService service.KeywordsService
}

func (h *KeywordsHandler) GetKeywordsListSortedByPopularity(ctx context.Context, _ *http.Request, _ any) (*service.KeywordsList, error) {
	return h.keywordsService.GetKeywordsListSortedByPopularity(ctx)
}

func (h *KeywordsHandler) UpdateKeywords(ctx context.Context, _ *http.Request, input service.UpdateKeywordsDto) (*transport.Empty, error) {
	user := auth.GetUser(ctx)

	err := h.keywordsService.UpdateKeywords(ctx, user, input)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func NewKeywordsHandler(service service.KeywordsService) *KeywordsHandler {
	return &KeywordsHandler{keywordsService: service}
}
