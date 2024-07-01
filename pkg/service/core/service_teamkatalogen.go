package core

import (
	"context"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TeamKatalogenService = &teamkatalogenService{}

type teamkatalogenService struct {
	teamKatalogenAPI service.TeamKatalogenAPI
}

func (t *teamkatalogenService) SearchTeamKatalogen(ctx context.Context, gcpGroups []string) ([]service.TeamkatalogenResult, error) {
	const op = "teamkatalogenService.SearchTeamKatalogen"

	res, err := t.teamKatalogenAPI.Search(ctx, gcpGroups)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return res, nil
}

func NewTeamKatalogenService(api service.TeamKatalogenAPI) *teamkatalogenService {
	return &teamkatalogenService{
		teamKatalogenAPI: api,
	}
}
