package handlers

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type TeamkatalogenHandler struct {
	teamKatalogenService service.TeamKatalogenService
}

func (h *TeamkatalogenHandler) SearchTeamKatalogen(ctx context.Context, r *http.Request, _ any) ([]service.TeamkatalogenResult, error) {
	const op errs.Op = "TeamkatalogenHandler.SearchTeamKatalogen"

	groups := r.URL.Query()["gcpGroups"]

	teams, err := h.teamKatalogenService.SearchTeamKatalogen(ctx, groups)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return teams, nil
}

func NewTeamKatalogenHandler(s service.TeamKatalogenService) *TeamkatalogenHandler {
	return &TeamkatalogenHandler{teamKatalogenService: s}
}
