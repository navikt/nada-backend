package handlers

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/errs"

	"github.com/navikt/nada-backend/pkg/service"
)

type PollyHandler struct {
	pollyService service.PollyService
}

func (h *PollyHandler) SearchPolly(ctx context.Context, r *http.Request, _ any) ([]*service.QueryPolly, error) {
	const op errs.Op = "PollyHandler.SearchPolly"

	result, err := h.pollyService.SearchPolly(ctx, r.URL.Query().Get("query"))
	if err != nil {
		return nil, errs.E(op, err)
	}

	return result, nil
}

func NewPollyHandler(s service.PollyService) *PollyHandler {
	return &PollyHandler{pollyService: s}
}
