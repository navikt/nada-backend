package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TeamKatalogenAPI = &teamKatalogenCache{}

type teamKatalogenCache struct {
	api   service.TeamKatalogenAPI
	cache cache.Cacher
}

func (t teamKatalogenCache) GetTeam(ctx context.Context, teamID uuid.UUID) (*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenCache.GetTeam"

	key := fmt.Sprintf("teamkatalogen:team:%s", teamID.String())

	team := &service.TeamkatalogenTeam{}
	valid := t.cache.Get(key, team)
	if valid {
		return team, nil
	}

	team, err := t.api.GetTeam(ctx, teamID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	t.cache.Set(key, team)

	return team, nil
}

func (t teamKatalogenCache) GetTeamCatalogURL(teamID uuid.UUID) string {
	return t.api.GetTeamCatalogURL(teamID)
}

func (t teamKatalogenCache) GetTeamsInProductArea(ctx context.Context, paID uuid.UUID) ([]*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenCache.GetTeamsInProductArea"

	key := fmt.Sprintf("teamkatalogen:teams:pa:%s", paID.String())

	teams := []*service.TeamkatalogenTeam{}
	valid := t.cache.Get(key, &teams)
	if valid {
		return teams, nil
	}

	teams, err := t.api.GetTeamsInProductArea(ctx, paID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	t.cache.Set(key, teams)

	return teams, nil
}

func (t teamKatalogenCache) GetProductAreas(ctx context.Context) ([]*service.TeamkatalogenProductArea, error) {
	const op errs.Op = "teamKatalogenCache.GetProductAreas"

	key := "teamkatalogen:productareas"

	pas := []*service.TeamkatalogenProductArea{}
	valid := t.cache.Get(key, &pas)
	if valid {
		return pas, nil
	}

	pas, err := t.api.GetProductAreas(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	t.cache.Set(key, pas)

	return pas, nil
}

func (t teamKatalogenCache) Search(ctx context.Context, gcpGroups []string) ([]service.TeamkatalogenResult, error) {
	const op = "teamKatalogenCache.Search"

	key := fmt.Sprintf("teamkatalogen:search:%v", gcpGroups)

	results := []service.TeamkatalogenResult{}
	valid := t.cache.Get(key, &results)
	if valid {
		return results, nil
	}

	results, err := t.api.Search(ctx, gcpGroups)
	if err != nil {
		return results, errs.E(op, err)
	}

	t.cache.Set(key, results)

	return results, nil
}

func NewTeamKatalogenCache(api service.TeamKatalogenAPI, cache cache.Cacher) *teamKatalogenCache {
	return &teamKatalogenCache{
		api:   api,
		cache: cache,
	}
}
