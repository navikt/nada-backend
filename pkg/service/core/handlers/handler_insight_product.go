package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type insightProductHandler struct {
	service service.InsightProductService
}

func (h *insightProductHandler) GetInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductHandler.GetInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	insightProduct, err := h.service.GetInsightProduct(ctx, id)
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
	const op errs.Op = "insightProductHandler.UpdateInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	insightProduct, err := h.service.UpdateInsightProduct(ctx, id, in)
	if err != nil {
		return nil, err
	}

	return insightProduct, nil
}

func (h *insightProductHandler) DeleteInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	const op errs.Op = "insightProductHandler.DeleteInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	deleted, err := h.service.DeleteInsightProduct(ctx, id)
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
