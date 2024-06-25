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

type joinableViewsHandler struct {
	service service.JoinableViewsService
}

// FIXME: return something other than a string
func (h *joinableViewsHandler) CreateJoinableViews(ctx context.Context, _ *http.Request, in service.NewJoinableViews) (string, error) {
	id, err := h.service.CreateJoinableViews(ctx, in)
	if err != nil {
		return "", nil
	}

	return id, nil
}

func (h *joinableViewsHandler) GetJoinableViewsForUser(ctx context.Context, _ *http.Request, _ any) ([]service.JoinableView, error) {
	views, err := h.service.GetJoinableViewsForUser(ctx)
	if err != nil {
		return nil, err
	}

	return views, nil
}

func (h *joinableViewsHandler) GetJoinableView(ctx context.Context, _ *http.Request, _ any) (*service.JoinableViewWithDatasource, error) {
	const op errs.Op = "joinableViewsHandler.GetJoinableView"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	view, err := h.service.GetJoinableView(ctx, id)
	if err != nil {
		return nil, err
	}

	return view, nil
}

func NewJoinableViewsHandler(service service.JoinableViewsService) *joinableViewsHandler {
	return &joinableViewsHandler{service: service}
}
