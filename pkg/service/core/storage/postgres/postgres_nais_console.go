package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.NaisConsoleStorage = &naisConsoleStorage{}

type naisConsoleStorage struct {
	db *database.Repo
}

func (s *naisConsoleStorage) GetAllTeamProjects(ctx context.Context) (map[string]string, error) {
	const op errs.Op = "naisConsoleStorage.GetAllTeamProjects"

	raw, err := s.db.Querier.GetTeamProjects(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	teamProjects := map[string]string{}
	for _, teamProject := range raw {
		teamProjects[teamProject.Team] = teamProject.Project
	}

	return teamProjects, nil
}

func (s *naisConsoleStorage) UpdateAllTeamProjects(ctx context.Context, teams map[string]string) error {
	const op errs.Op = "naisConsoleStorage.UpdateAllTeamProjects"

	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := s.db.Querier.WithTx(tx)

	err = q.ClearTeamProjectsCache(ctx)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	for team, project := range teams {
		_, err := q.AddTeamProject(ctx, gensql.AddTeamProjectParams{
			Team:    team,
			Project: project,
		})
		if err != nil {
			return errs.E(errs.Database, op, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *naisConsoleStorage) GetTeamProject(ctx context.Context, naisTeam string) (string, error) {
	const op errs.Op = "naisConsoleStorage.GetTeamProject"

	if strings.Contains(naisTeam, "@") {
		naisTeam = strings.Split(naisTeam, "@")[0]
	}

	teamProjects, err := s.GetAllTeamProjects(ctx)
	if err != nil {
		return "", errs.E(op, err)
	}

	project, ok := teamProjects[naisTeam]
	if !ok {
		return "", errs.E(errs.NotExist, op, errs.Parameter("naisTeam"), fmt.Errorf("team %s not found", naisTeam))
	}

	return project, nil
}

func NewNaisConsoleStorage(db *database.Repo) *naisConsoleStorage {
	return &naisConsoleStorage{
		db: db,
	}
}
