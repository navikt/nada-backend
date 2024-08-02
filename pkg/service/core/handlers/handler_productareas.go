package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"

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

func (h *ProductAreasHandler) GetProductAreaWithAssets(ctx context.Context, _ *http.Request, _ any) (*service.ProductAreaWithAssets, error) {
	const op errs.Op = "ProductAreasHandler.GetProductAreaWithAssets"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	pa, err := h.service.GetProductAreaWithAssets(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return pa, nil
}

func NewProductAreasHandler(service service.ProductAreaService) *ProductAreasHandler {
	return &ProductAreasHandler{service: service}
}
