package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type JoinableViewsHandler struct {
	service service.JoinableViewsService
}

// FIXME: return something other than a string
func (h *JoinableViewsHandler) CreateJoinableViews(ctx context.Context, _ *http.Request, in service.NewJoinableViews) (string, error) {
	const op errs.Op = "JoinableViewsHandler.CreateJoinableViews"

	user := auth.GetUser(ctx)
	if user == nil {
		return "", errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	id, err := h.service.CreateJoinableViews(ctx, user, in)
	if err != nil {
		return "", errs.E(op, err)
	}

	return id, nil
}

func (h *JoinableViewsHandler) GetJoinableViewsForUser(ctx context.Context, _ *http.Request, _ any) ([]service.JoinableView, error) {
	const op errs.Op = "JoinableViewsHandler.GetJoinableViewsForUser"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	views, err := h.service.GetJoinableViewsForUser(ctx, user)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return views, nil
}

func (h *JoinableViewsHandler) GetJoinableView(ctx context.Context, _ *http.Request, _ any) (*service.JoinableViewWithDatasource, error) {
	const op errs.Op = "JoinableViewsHandler.GetJoinableView"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	view, err := h.service.GetJoinableView(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return view, nil
}

func NewJoinableViewsHandler(service service.JoinableViewsService) *JoinableViewsHandler {
	return &JoinableViewsHandler{service: service}
}
