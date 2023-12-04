package database

import (
	"context"
	"database/sql"

	"github.com/navikt/nada-backend/pkg/database/gensql"
)

func (r *Repo) GetTeamAndProductAreaID(ctx context.Context, teamID string) (gensql.TeamProductareaMapping, error) {
	return r.querier.GetTeamAndProductAreaID(ctx, teamID)
}

func (r *Repo) GetTeamsAndProductAreaIDs(ctx context.Context) ([]gensql.TeamProductareaMapping, error) {
	return r.querier.GetTeamsAndProductAreaIDs(ctx)
}

func (r *Repo) UpdateProductAreaForTeam(ctx context.Context, teamID, productAreaID string) error {
	return r.querier.UpdateProductAreaForTeam(ctx, gensql.UpdateProductAreaForTeamParams{
		TeamID:        teamID,
		ProductAreaID: sql.NullString{String: productAreaID, Valid: true},
	})
}
