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

func (r *Repo) AddTeamProject(ctx context.Context, team, projectID string) (*gensql.TeamProject, error) {
	teamProject, err := r.querier.AddTeamProject(ctx, gensql.AddTeamProjectParams{
		Team:    team,
		Project: projectID,
	})
	if err != nil {
		return nil, err
	}
	return &teamProject, nil
}
