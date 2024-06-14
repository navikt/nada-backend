package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type keywordsHandler struct {
	keywordsService service.KeywordsService
}

func (h *keywordsHandler) GetKeywordsListSortedByPopularity(ctx context.Context, _ *http.Request, _ any) (*service.KeywordsList, error) {
	return h.keywordsService.GetKeywordsListSortedByPopularity(ctx)
}

func (h *keywordsHandler) UpdateKeywords(ctx context.Context, _ *http.Request, input service.UpdateKeywordsDto) (*Empty, error) {
	err := h.keywordsService.UpdateKeywords(ctx, input)
	if err != nil {
		return nil, err
	}

	return &Empty{}, nil
}

func NewKeywordsHandler(service service.KeywordsService) *keywordsHandler {
	return &keywordsHandler{keywordsService: service}
}
