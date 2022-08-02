package database

import (
	"context"

	"github.com/navikt/nada-backend/pkg/database/gensql"
)

func (r *Repo) GetTeamProjects(ctx context.Context) ([]gensql.TeamProject, error) {
	teamProjects, err := r.querier.GetTeamProjects(ctx)
	if err != nil {
		return nil, err
	}
	return teamProjects, nil
}

func (r *Repo) UpdateTeamProjectsCache(ctx context.Context, teamProjects map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	querier := r.querier.WithTx(tx)

	if err := querier.ClearTeamProjectsCache(ctx); err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back clear projects cache transaction")
		}
		return err
	}

	for team, projectID := range teamProjects {
		_, err := querier.AddTeamProject(ctx, gensql.AddTeamProjectParams{
			Team:    team,
			Project: projectID,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back update projects cache transaction")
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
