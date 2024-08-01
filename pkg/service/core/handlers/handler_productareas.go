package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type ProductAreasHandler struct {
	service service.ProductAreaService
}

func (h *ProductAreasHandler) GetProductAreas(ctx context.Context, _ *http.Request, _ any) (*service.ProductAreasDto, error) {
	const op errs.Op = "ProductAreasHandler.GetProductAreas"

	p, err := h.service.GetProductAreas(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return p, nil
}

func (h *ProductAreasHandler) GetProductAreaWithAssets(ctx context.Context, r *http.Request, _ any) (*service.ProductAreaWithAssets, error) {
	const op errs.Op = "ProductAreasHandler.GetProductAreaWithAssets"

	id, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	return h.service.GetProductAreaWithAssets(ctx, id)
}

func NewProductAreasHandler(service service.ProductAreaService) *ProductAreasHandler {
	return &ProductAreasHandler{service: service}
}
