package handlers

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type MetabaseHandler struct {
	service service.MetabaseService
}

func (h *MetabaseHandler) MapDataset(ctx context.Context, _ *http.Request, in service.DatasetMap) (*service.Dataset, error) {
	const op errs.Op = "MetabaseHandler.MapDataset"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	user := auth.GetUser(ctx)

	dataset, err := h.service.MapDataset(ctx, user, id, in.Services)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func NewMetabaseHandler(service service.MetabaseService) *MetabaseHandler {
	return &MetabaseHandler{
		service: service,
	}
}
