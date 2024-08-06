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
	"github.com/navikt/nada-backend/pkg/service/core/transport"
)

type AccessHandler struct {
	accessService   service.AccessService
	metabaseService service.MetabaseService
	gcpProjectID    string
}

func (h *AccessHandler) RevokeAccessToDataset(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.RevokeAccessToDataset"

	id, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err = h.metabaseService.RevokeMetabaseAccessFromAccessID(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	err = h.accessService.RevokeAccessToDataset(ctx, user, id, h.gcpProjectID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *AccessHandler) GrantAccessToDataset(ctx context.Context, _ *http.Request, in service.GrantAccessData) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.GrantAccessToDataset"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err := h.accessService.GrantAccessToDataset(ctx, user, in, h.gcpProjectID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	err = h.metabaseService.GrantMetabaseAccess(ctx, in.DatasetID, *in.Subject, *in.SubjectType)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *AccessHandler) GetAccessRequests(ctx context.Context, r *http.Request, _ interface{}) (*service.AccessRequestsWrapper, error) {
	op := "AccessHandler.GetAccessRequests"

	id, err := uuid.Parse(r.URL.Query().Get("datasetId"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	access, err := h.accessService.GetAccessRequests(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return access, nil
}

func (h *AccessHandler) ProcessAccessRequest(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.ProcessAccessRequest"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	reason := r.URL.Query().Get("reason")
	action := r.URL.Query().Get("action")

	switch action {
	case "approve":
		err := h.accessService.ApproveAccessRequest(ctx, user, id)
		if err != nil {
			return nil, errs.E(op, err)
		}

		return &transport.Empty{}, nil
	case "deny":
		err := h.accessService.DenyAccessRequest(ctx, user, id, &reason)
		if err != nil {
			return nil, errs.E(op, err)
		}

		return &transport.Empty{}, nil
	default:
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("invalid action: %s", action))
	}
}

func (h *AccessHandler) NewAccessRequest(ctx context.Context, _ *http.Request, in service.NewAccessRequestDTO) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.NewAccessRequest"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err := h.accessService.CreateAccessRequest(ctx, user, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *AccessHandler) DeleteAccessRequest(ctx context.Context, _ *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.DeleteAccessRequest"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err = h.accessService.DeleteAccessRequest(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *AccessHandler) UpdateAccessRequest(ctx context.Context, _ *http.Request, in service.UpdateAccessRequestDTO) (*transport.Empty, error) {
	const op errs.Op = "AccessHandler.UpdateAccessRequest"

	// FIXME: should we verify the user here
	err := h.accessService.UpdateAccessRequest(ctx, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func NewAccessHandler(
	service service.AccessService,
	metabaseService service.MetabaseService,
	gcpProjectID string,
) *AccessHandler {
	return &AccessHandler{
		accessService:   service,
		metabaseService: metabaseService,
		gcpProjectID:    gcpProjectID,
	}
}
