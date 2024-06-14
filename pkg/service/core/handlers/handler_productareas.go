package handlers

import (
	"context"
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
	return h.service.GetProductAreaWithAssets(ctx, r.URL.Query().Get("id"))
}

func NewProductAreasHandler(service service.ProductAreaService) *productAreasHandler {
	return &productAreasHandler{service: service}
}
