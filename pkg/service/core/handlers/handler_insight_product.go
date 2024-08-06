package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type InsightProductHandler struct {
	service service.InsightProductService
}

func (h *InsightProductHandler) GetInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	const op errs.Op = "InsightProductHandler.GetInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	insightProduct, err := h.service.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return insightProduct, nil
}

func (h *InsightProductHandler) CreateInsightProduct(ctx context.Context, _ *http.Request, in service.NewInsightProduct) (*service.InsightProduct, error) {
	const op errs.Op = "InsightProductHandler.CreateInsightProduct"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err := user.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	insightProduct, err := h.service.CreateInsightProduct(ctx, user, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return insightProduct, nil
}

func (h *InsightProductHandler) UpdateInsightProduct(ctx context.Context, _ *http.Request, in service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	const op errs.Op = "InsightProductHandler.UpdateInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	insightProduct, err := h.service.UpdateInsightProduct(ctx, user, id, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return insightProduct, nil
}

func (h *InsightProductHandler) DeleteInsightProduct(ctx context.Context, _ *http.Request, _ any) (*service.InsightProduct, error) {
	const op errs.Op = "InsightProductHandler.DeleteInsightProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	deleted, err := h.service.DeleteInsightProduct(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return deleted, nil
}

func NewInsightProductHandler(service service.InsightProductService) *InsightProductHandler {
	return &InsightProductHandler{
		service: service,
	}
}
