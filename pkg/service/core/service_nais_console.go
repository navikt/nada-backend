package core

import (
	"context"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.NaisConsoleService = &naisConsoleService{}

type naisConsoleService struct {
	storage service.NaisConsoleStorage
	api     service.NaisConsoleAPI
}

func (s *naisConsoleService) UpdateAllTeamProjects(ctx context.Context) error {
	const op errs.Op = "naisConsoleService.UpdateAllTeamProjects"

	projects, err := s.api.GetGoogleProjectsForAllTeams(ctx)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.storage.UpdateAllTeamProjects(ctx, projects)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func NewNaisConsoleService(storage service.NaisConsoleStorage, api service.NaisConsoleAPI) *naisConsoleService {
	return &naisConsoleService{
		storage: storage,
		api:     api,
	}
}
