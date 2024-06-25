package handlers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type productAreasHandler struct {
	service service.ProductAreaService
}

func (h *productAreasHandler) GetProductAreas(ctx context.Context, _ *http.Request, _ any) (*service.ProductAreasDto, error) {
	return h.service.GetProductAreas(ctx)
}

func (h *productAreasHandler) GetProductAreaWithAssets(ctx context.Context, r *http.Request, _ any) (*service.ProductAreaWithAssets, error) {
	const op errs.Op = "productAreasHandler.GetProductAreaWithAssets"

	id, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	return h.service.GetProductAreaWithAssets(ctx, id)
}

func NewProductAreasHandler(service service.ProductAreaService) *productAreasHandler {
	return &productAreasHandler{service: service}
}
