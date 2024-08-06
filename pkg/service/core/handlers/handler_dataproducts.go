package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
)

type DataProductsHandler struct {
	service service.DataProductsService
}

func (h *DataProductsHandler) GetDataProduct(ctx context.Context, _ *http.Request, _ any) (*service.DataproductWithDataset, error) {
	const op errs.Op = "DataProductsHandler.GetDataProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	dp, err := h.service.GetDataproduct(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dp, nil
}

func (h *DataProductsHandler) CreateDataProduct(ctx context.Context, _ *http.Request, in service.NewDataproduct) (*service.DataproductMinimal, error) {
	const op errs.Op = "DataProductsHandler.CreateDataProduct"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	dp, err := h.service.CreateDataproduct(ctx, user, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dp, nil
}

func (h *DataProductsHandler) DeleteDataProduct(ctx context.Context, _ *http.Request, _ interface{}) (*transport.Empty, error) {
	const op errs.Op = "DataProductsHandler.DeleteDataProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	_, err = h.service.DeleteDataproduct(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	// FIXME: it might be wrong to return empty, since the response is not empty
	return &transport.Empty{}, nil
}

func (h *DataProductsHandler) UpdateDataProduct(ctx context.Context, _ *http.Request, in service.UpdateDataproductDto) (*service.DataproductMinimal, error) {
	const op errs.Op = "DataProductsHandler.UpdateDataProduct"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	dp, err := h.service.UpdateDataproduct(ctx, user, id, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dp, nil
}

func (h *DataProductsHandler) GetDatasetsMinimal(ctx context.Context, _ *http.Request, _ interface{}) ([]*service.DatasetMinimal, error) {
	const op errs.Op = "DataProductsHandler.GetDatasetsMinimal"

	datasets, err := h.service.GetDatasetsMinimal(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return datasets, nil
}

func (h *DataProductsHandler) GetDataset(ctx context.Context, _ *http.Request, _ interface{}) (*service.Dataset, error) {
	const op errs.Op = "DataProductsHandler.GetDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	dataset, err := h.service.GetDataset(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return dataset, nil
}

func (h *DataProductsHandler) CreateDataset(ctx context.Context, _ *http.Request, in service.NewDataset) (*string, error) {
	const op errs.Op = "DataProductsHandler.CreateDataset"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	ds, err := h.service.CreateDataset(ctx, user, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	// FIXME: is it correct to just return the slug here?
	return &ds.Slug, nil
}

func (h *DataProductsHandler) UpdateDataset(ctx context.Context, _ *http.Request, in service.UpdateDatasetDto) (string, error) {
	const op errs.Op = "DataProductsHandler.UpdateDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return "", errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return "", errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	dataset, err := h.service.UpdateDataset(ctx, user, id, in)
	if err != nil {
		return "", errs.E(op, err)
	}

	return dataset, nil
}

func (h *DataProductsHandler) DeleteDataset(ctx context.Context, _ *http.Request, _ interface{}) (*transport.Empty, error) {
	const op errs.Op = "DataProductsHandler.DeleteDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	_, err = h.service.DeleteDataset(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *DataProductsHandler) GetAccessiblePseudoDatasetsForUser(ctx context.Context, _ *http.Request, _ interface{}) ([]*service.PseudoDataset, error) {
	const op errs.Op = "DataProductsHandler.GetAccessiblePseudoDatasetsForUser"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	datasets, err := h.service.GetAccessiblePseudoDatasetsForUser(ctx, user)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return datasets, nil
}

func NewDataProductsHandler(s service.DataProductsService) *DataProductsHandler {
	return &DataProductsHandler{
		service: s,
	}
}
