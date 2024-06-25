package handlers

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"net/http"
)

type accessHandler struct {
	accessService   service.AccessService
	metabaseService service.MetabaseService
	gcpProjectID    string
}

func (h *accessHandler) RevokeAccessToDataset(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "accessHandler.RevokeAccessToDataset"

	id, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	user := auth.GetUser(ctx)

	err = h.metabaseService.RevokeMetabaseAccessFromAccessID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = h.accessService.RevokeAccessToDataset(ctx, user, id, h.gcpProjectID)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *accessHandler) GrantAccessToDataset(ctx context.Context, _ *http.Request, in service.GrantAccessData) (*transport.Empty, error) {
	user := auth.GetUser(ctx)

	err := h.accessService.GrantAccessToDataset(ctx, user, in, h.gcpProjectID)
	if err != nil {
		return nil, err
	}

	err = h.metabaseService.GrantMetabaseAccess(ctx, user, in.DatasetID, *in.Subject, *in.SubjectType)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *accessHandler) GetAccessRequests(ctx context.Context, r *http.Request, _ interface{}) (*service.AccessRequestsWrapper, error) {
	op := "accessHandler.GetAccessRequests"

	id, err := uuid.Parse(r.URL.Query().Get("datasetId"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	access, err := h.accessService.GetAccessRequests(ctx, id)
	if err != nil {
		return nil, err
	}

	return access, nil
}

func (h *accessHandler) ProcessAccessRequest(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "accessHandler.ProcessAccessRequest"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	user := auth.GetUser(ctx)

	reason := r.URL.Query().Get("reason")
	action := r.URL.Query().Get("action")

	switch action {
	case "approve":
		return &transport.Empty{}, h.accessService.ApproveAccessRequest(ctx, user, id)
	case "deny":
		return &transport.Empty{}, h.accessService.DenyAccessRequest(ctx, user, id, &reason)
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}
}

func (h *accessHandler) NewAccessRequest(ctx context.Context, _ *http.Request, in service.NewAccessRequestDTO) (*transport.Empty, error) {
	user := auth.GetUser(ctx)

	err := h.accessService.CreateAccessRequest(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *accessHandler) DeleteAccessRequest(ctx context.Context, _ *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "accessHandler.DeleteAccessRequest"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing access request id: %w", err))
	}

	user := auth.GetUser(ctx)

	err = h.accessService.DeleteAccessRequest(ctx, user, id)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *accessHandler) UpdateAccessRequest(ctx context.Context, _ *http.Request, in service.UpdateAccessRequestDTO) (*transport.Empty, error) {
	// FIXME: should we verify the user here

	err := h.accessService.UpdateAccessRequest(ctx, in)
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func NewAccessHandler(
	service service.AccessService,
	metabaseService service.MetabaseService,
	gcpProjectID string,
) *accessHandler {
	return &accessHandler{
		accessService:   service,
		metabaseService: metabaseService,
		gcpProjectID:    gcpProjectID,
	}
}
