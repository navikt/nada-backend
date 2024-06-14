package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type insightProductHandler struct {
	service service.InsightProductService
}

func (h *insightProductHandler) GetInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	insightProduct, err := h.service.GetInsightProduct(ctx, chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, err
	}

	return insightProduct, nil
}

func (h *insightProductHandler) CreateInsightProduct(ctx context.Context, _ *http.Request, in service.NewInsightProduct) (*service.InsightProduct, error) {
	insightProduct, err := h.service.CreateInsightProduct(ctx, in)
	if err != nil {
		return nil, err
	}

	return insightProduct, nil
}

func (h *insightProductHandler) UpdateInsightProduct(ctx context.Context, _ *http.Request, in service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	insightProduct, err := h.service.UpdateInsightProduct(ctx, chi.URLParamFromCtx(ctx, "id"), in)
	if err != nil {
		return nil, err
	}

	return insightProduct, nil
}

func (h *insightProductHandler) DeleteInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	deleted, err := h.service.DeleteInsightProduct(ctx, chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, err
	}

	return deleted, nil
}

func NewInsightProductHandler(service service.InsightProductService) *insightProductHandler {
	return &insightProductHandler{
		service: service,
	}
}
