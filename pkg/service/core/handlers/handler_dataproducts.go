package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type dataProductsHandler struct {
	service service.DataProductsService
}

func (h *dataProductsHandler) GetDataProduct(ctx context.Context, r *http.Request, _ interface{}) (*service.DataproductWithDataset, error) {
	dp, err := h.service.GetDataproduct(ctx, chi.URLParam(r, "id"))
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func (h *dataProductsHandler) CreateDataProduct(ctx context.Context, _ *http.Request, in service.NewDataproduct) (*service.DataproductMinimal, error) {
	dp, err := h.service.CreateDataproduct(ctx, in)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func (h *dataProductsHandler) DeleteDataProduct(ctx context.Context, r *http.Request, _ interface{}) (*Empty, error) {
	_, err := h.service.DeleteDataproduct(ctx, chi.URLParam(r, "id"))
	if err != nil {
		return nil, err
	}

	// FIXME: it might be wrong to return empty, since the response is not empty
	return &Empty{}, nil
}

func (h *dataProductsHandler) UpdateDataProduct(ctx context.Context, r *http.Request, in service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	dp, err := h.service.UpdateDataproduct(ctx, chi.URLParam(r, "id"), in)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func NewDataProductsHandler(s service.DataProductsService) *dataProductsHandler {
	return &dataProductsHandler{
		service: s,
	}
}
