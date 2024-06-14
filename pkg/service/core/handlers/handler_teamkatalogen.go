package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type teamkatalogenHandler struct {
	teamKatalogenService service.TeamKatalogenService
}

func (h *teamkatalogenHandler) SearchTeamKatalogen(ctx context.Context, r *http.Request, _ any) ([]service.TeamkatalogenResult, error) {
	return h.teamKatalogenService.SearchTeamKatalogen(ctx, r.URL.Query()["gcpGroups"])
}

func NewTeamKatalogenHandler(s service.TeamKatalogenService) *teamkatalogenHandler {
	return &teamkatalogenHandler{teamKatalogenService: s}
}
