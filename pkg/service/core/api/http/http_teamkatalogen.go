package http

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ service.TeamKatalogenAPI = &teamKatalogenAPI{}

type teamKatalogenAPI struct {
	fetcher tk.Fetcher
	log     zerolog.Logger
}

func (t *teamKatalogenAPI) GetProductAreas(ctx context.Context) ([]*service.TeamkatalogenProductArea, error) {
	const op errs.Op = "teamKatalogenAPI.GetProductAreas"

	productAreas, err := t.fetcher.GetProductAreas(ctx)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	pas := make([]*service.TeamkatalogenProductArea, 0)
	for _, pa := range productAreas.Content {
		pas = append(pas, &service.TeamkatalogenProductArea{
			ID:       pa.ID,
			Name:     pa.Name,
			AreaType: pa.AreaType,
		})
	}

	return pas, nil
}

func (t *teamKatalogenAPI) GetTeam(ctx context.Context, teamID uuid.UUID) (*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenAPI.GetTeam"

	team, err := t.fetcher.GetTeam(ctx, teamID)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	return &service.TeamkatalogenTeam{
		ID:            team.ID,
		Name:          team.Name,
		ProductAreaID: team.ProductAreaID,
	}, nil
}

func (t *teamKatalogenAPI) GetTeamCatalogURL(teamID uuid.UUID) string {
	return t.fetcher.GetTeamCatalogURL(teamID)
}

func (t *teamKatalogenAPI) GetTeamsInProductArea(ctx context.Context, paID uuid.UUID) ([]*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenAPI.GetTeamsInProductArea"

	teams, err := t.fetcher.GetTeamsInProductArea(ctx, paID)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	teamsGraph := make([]*service.TeamkatalogenTeam, len(teams.Content))
	for idx, t := range teams.Content {
		teamsGraph[idx] = &service.TeamkatalogenTeam{
			ID:            t.ID,
			Name:          t.Name,
			ProductAreaID: t.ProductAreaID,
		}
	}

	return teamsGraph, nil
}

func (t *teamKatalogenAPI) Search(ctx context.Context, gcpGroups []string) ([]service.TeamkatalogenResult, error) {
	const op errs.Op = "teamKatalogenAPI.Search"

	teams, err := t.fetcher.GetTeams(ctx)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	log.Info().Msgf("matching against groups %v", gcpGroups)

	var ret []service.TeamkatalogenResult
	for _, r := range teams.Content {
		isMatch := false
		if ContainsAnyCaseInsensitive(r.Name, gcpGroups) {
			isMatch = true
		}
		for _, team := range r.NaisTeams {
			if ContainsAnyCaseInsensitive(team, gcpGroups) {
				isMatch = true
				break
			}
		}

		if isMatch {
			ret = append(ret, service.TeamkatalogenResult{
				URL:           r.Links.UI,
				Name:          r.Name,
				Description:   r.Description,
				ProductAreaID: r.ProductAreaID.String(),
				TeamID:        r.ID.String(),
			})
		}
	}

	return ret, nil
}

func ContainsAnyCaseInsensitive(s string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, q := range patterns {
		if strings.Contains(strings.ToLower(s), strings.ToLower(q)) {
			return true
		}
	}
	return false
}

func NewTeamKatalogenAPI(fetcher tk.Fetcher, log zerolog.Logger) *teamKatalogenAPI {
	return &teamKatalogenAPI{
		fetcher: fetcher,
		log:     log,
	}
}
