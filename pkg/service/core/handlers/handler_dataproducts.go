package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"net/http"
)

type dataProductsHandler struct {
	service service.DataProductsService
}

func (h *dataProductsHandler) GetDataProduct(ctx context.Context, _ *http.Request, _ any) (*service.DataproductWithDataset, error) {
	dp, err := h.service.GetDataproduct(ctx, chi.URLParamFromCtx(ctx, "id"))
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

func (h *dataProductsHandler) DeleteDataProduct(ctx context.Context, _ *http.Request, _ interface{}) (*transport.Empty, error) {
	_, err := h.service.DeleteDataproduct(ctx, chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, err
	}

	// FIXME: it might be wrong to return empty, since the response is not empty
	return &transport.Empty{}, nil
}

func (h *dataProductsHandler) UpdateDataProduct(ctx context.Context, _ *http.Request, in service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	dp, err := h.service.UpdateDataproduct(ctx, chi.URLParamFromCtx(ctx, "id"), in)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func (h *dataProductsHandler) GetDatasetsMinimal(ctx context.Context, _ *http.Request, _ interface{}) ([]*service.DatasetMinimal, error) {
	datasets, err := h.service.GetDatasetsMinimal(ctx)
	if err != nil {
		return nil, err
	}

	return datasets, nil
}

func (h *dataProductsHandler) GetDataset(ctx context.Context, _ *http.Request, _ interface{}) (*service.Dataset, error) {
	dataset, err := h.service.GetDataset(ctx, chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func (h *dataProductsHandler) CreateDataset(ctx context.Context, _ *http.Request, in service.NewDataset) (*string, error) {
	datasetSlug, err := h.service.CreateDataset(ctx, in)
	if err != nil {
		return nil, err
	}

	// FIXME: is it correct to just return the slug here?
	return datasetSlug, nil
}

func (h *dataProductsHandler) UpdateDataset(ctx context.Context, _ *http.Request, in service.UpdateDatasetDto) (string, error) {
	dataset, err := h.service.UpdateDataset(ctx, chi.URLParamFromCtx(ctx, "id"), in)
	if err != nil {
		return "", err
	}

	return dataset, nil
}

func (h *dataProductsHandler) DeleteDataset(ctx context.Context, _ *http.Request, _ interface{}) (*transport.Empty, error) {
	_, err := h.service.DeleteDataset(ctx, chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *dataProductsHandler) GetAccessiblePseudoDatasetsForUser(ctx context.Context, _ *http.Request, _ interface{}) ([]*service.PseudoDataset, error) {
	datasets, err := h.service.GetAccessiblePseudoDatasetsForUser(ctx)
	if err != nil {
		return nil, err
	}

	return datasets, nil
}

func NewDataProductsHandler(s service.DataProductsService) *dataProductsHandler {
	return &dataProductsHandler{
		service: s,
	}
}
