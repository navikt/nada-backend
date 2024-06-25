package handlers

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type metabaseHandler struct {
	service service.MetabaseService
}

func (h *metabaseHandler) MapDataset(ctx context.Context, _ *http.Request, in service.DatasetMap) (*service.Dataset, error) {
	const op errs.Op = "metabaseHandler.MapDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	dataset, err := h.service.MapDataset(ctx, id, in.Services)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func NewMetabaseHandler(service service.MetabaseService) *metabaseHandler {
	return &metabaseHandler{
		service: service,
	}
}
