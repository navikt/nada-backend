package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.ProductAreaStorage = &productAreaStorage{}

type productAreaStorage struct {
	db *database.Repo
}

func (s *productAreaStorage) UpsertProductAreaAndTeam(ctx context.Context, pas []*service.UpsertProductAreaRequest, teams []*service.UpsertTeamRequest) error {
	const op errs.Op = "productAreaStorage.UpsertProductAreaAndTeam"

	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	q := s.db.Querier.WithTx(tx)

	for _, pa := range pas {
		err = q.UpsertProductArea(ctx, gensql.UpsertProductAreaParams{
			ID: pa.ID,
			Name: sql.NullString{
				String: pa.Name,
				Valid:  true,
			},
		})
		if err != nil {
			return errs.E(errs.Database, op, err)
		}
	}

	for _, team := range teams {
		err = q.UpsertTeam(context.Background(), gensql.UpsertTeamParams{
			ID:            team.ID,
			ProductAreaID: uuid.NullUUID{UUID: team.ProductAreaID, Valid: true},
			Name: sql.NullString{
				String: team.Name,
				Valid:  true,
			},
		})
		if err != nil {
			return errs.E(errs.Database, op, err)
		}
	}

	tx.Commit()

	return nil
}

func (s *productAreaStorage) GetDashboard(ctx context.Context, id uuid.UUID) (*service.Dashboard, error) {
	const op errs.Op = "productAreaStorage.GetDashboard"

	dashboard, err := s.db.Querier.GetDashboard(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errs.E(errs.Database, op, err)
	}

	return &service.Dashboard{
		ID:  dashboard.ID,
		Url: dashboard.Url,
	}, nil
}

func (s *productAreaStorage) GetProductArea(ctx context.Context, paID uuid.UUID) (*service.ProductArea, error) {
	const op errs.Op = "productAreaStorage.GetProductArea"

	pa, err := s.db.Querier.GetProductArea(ctx, paID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err, errs.Parameter("product_area_id"))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	teams, err := s.db.Querier.GetTeamsInProductArea(ctx, uuid.NullUUID{
		UUID:  pa.ID,
		Valid: true,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errs.E(errs.Database, op, err)
	}

	paTeams := make([]*service.Team, 0)
	for _, team := range teams {
		paTeams = append(paTeams, &service.Team{
			TeamkatalogenTeam: &service.TeamkatalogenTeam{
				ID:            team.ID,
				Name:          team.Name.String,
				ProductAreaID: team.ProductAreaID.UUID,
			},
		})
	}

	areaType := ""
	if pa.AreaType.Valid {
		areaType = pa.AreaType.String
	}

	return &service.ProductArea{
		TeamkatalogenProductArea: &service.TeamkatalogenProductArea{
			ID:       pa.ID,
			Name:     pa.Name.String,
			AreaType: areaType,
		},
		Teams: paTeams,
	}, nil
}

func (s *productAreaStorage) GetProductAreas(ctx context.Context) ([]*service.ProductArea, error) {
	const op errs.Op = "productAreaStorage.GetProductAreas"

	pas, err := s.db.Querier.GetProductAreas(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	// FIXME: not optimal, but unsure how else to do this
	teams, err := s.db.Querier.GetAllTeams(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	productAreas := make([]*service.ProductArea, 0)
	for _, pa := range pas {
		paTeams := make([]*service.Team, 0)
		for _, team := range teams {
			if team.ProductAreaID.Valid && team.ProductAreaID.UUID == pa.ID {
				paTeams = append(paTeams, &service.Team{
					TeamkatalogenTeam: &service.TeamkatalogenTeam{
						ID:            team.ID,
						Name:          team.Name.String,
						ProductAreaID: team.ProductAreaID.UUID,
					},
				})
			}
		}
		areaType := ""
		if pa.AreaType.Valid {
			areaType = pa.AreaType.String
		}
		productAreas = append(productAreas, &service.ProductArea{
			TeamkatalogenProductArea: &service.TeamkatalogenProductArea{
				ID:       pa.ID,
				Name:     pa.Name.String,
				AreaType: areaType,
			},
			Teams:        paTeams,
			DashboardURL: "",
		})
	}

	return productAreas, nil
}

func NewProductAreaStorage(db *database.Repo) *productAreaStorage {
	return &productAreaStorage{db: db}
}
