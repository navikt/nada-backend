package handlers

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/service"
)

type PollyHandler struct {
	pollyService service.PollyService
}

func (h *PollyHandler) SearchPolly(ctx context.Context, r *http.Request, _ any) ([]*service.QueryPolly, error) {
	return h.pollyService.SearchPolly(ctx, r.URL.Query().Get("query"))
}

func NewPollyHandler(s service.PollyService) *PollyHandler {
	return &PollyHandler{pollyService: s}
}
