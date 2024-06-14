package core

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TeamKatalogenService = &teamkatalogenService{}

type teamkatalogenService struct {
	teamKatalogenAPI service.TeamKatalogenAPI
}

func (t *teamkatalogenService) SearchTeamKatalogen(ctx context.Context, gcpGroups []string) ([]service.TeamkatalogenResult, error) {
	return t.teamKatalogenAPI.Search(ctx, gcpGroups)
}

func NewTeamKatalogenService(api service.TeamKatalogenAPI) *teamkatalogenService {
	return &teamkatalogenService{teamKatalogenAPI: api}
}
