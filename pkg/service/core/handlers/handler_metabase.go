package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type metabaseHandler struct {
	service service.MetabaseService
}

func (h *metabaseHandler) MapDataset(ctx context.Context, _ *http.Request, in service.DatasetMap) (*service.Dataset, error) {
	dataset, err := h.service.MapDataset(ctx, chi.URLParamFromCtx(ctx, "id"), in.Services)
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
