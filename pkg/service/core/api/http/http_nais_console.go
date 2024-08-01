package http

import (
	"context"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/nc"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.NaisConsoleAPI = &naisConsoleAPI{}

type naisConsoleAPI struct {
	fetcher nc.Fetcher
}

func (a *naisConsoleAPI) GetGoogleProjectsForAllTeams(ctx context.Context) (map[string]string, error) {
	const op errs.Op = "naisConsoleAPI.GetGoogleProjectsForAllTeams"

	projects, err := a.fetcher.GetTeamGoogleProjects(ctx)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	return projects, nil
}

func NewNaisConsoleAPI(fetcher nc.Fetcher) *naisConsoleAPI {
	return &naisConsoleAPI{
		fetcher: fetcher,
	}
}
