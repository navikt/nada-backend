package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetDataproducts(ctx context.Context, limit, offset int) ([]*models.Dataproduct, error) {
	dataproducts := []*models.Dataproduct{}

	res, err := r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts from database: %w", err)
	}

	for _, entry := range res {
		dataproducts = append(dataproducts, dataproductFromSQL(entry))
	}

	return dataproducts, nil
}

func (r *Repo) GetDataproductsByUserAccess(ctx context.Context, user string) ([]*models.Dataproduct, error) {
	// todo: necessary?
	return nil, nil
}

func (r *Repo) GetDataproductsByGroups(ctx context.Context, groups []string) ([]*models.Dataproduct, error) {
	dps := []*models.Dataproduct{}

	res, err := r.querier.GetDataproductsByGroups(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts by group from database: %w", err)
	}

	for _, entry := range res {
		dps = append(dps, dataproductFromSQL(entry))
	}

	return dps, nil
}

func (r *Repo) GetDataproductByProductArea(ctx context.Context, paID string) ([]*models.Dataproduct, error) {
	dps, err := r.querier.GetDataproductsByProductArea(ctx, sql.NullString{String: paID, Valid: true})
	if err != nil {
		return nil, err
	}

	dpsGraph := make([]*models.Dataproduct, len(dps))
	for idx, dp := range dps {
		dpsGraph[idx] = dataproductFromSQL(dp)
	}

	return dpsGraph, nil
}

func (r *Repo) GetDataproductByTeam(ctx context.Context, teamID string) ([]*models.Dataproduct, error) {
	dps, err := r.querier.GetDataproductsByTeam(ctx, sql.NullString{String: teamID, Valid: true})
	if err != nil {
		return nil, err
	}

	dpsGraph := make([]*models.Dataproduct, len(dps))
	for idx, dp := range dps {
		dpsGraph[idx] = dataproductFromSQL(dp)
	}

	return dpsGraph, nil
}

func (r *Repo) GetDataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	res, err := r.querier.GetDataproduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) CreateDataproduct(ctx context.Context, dp models.NewDataproduct, user *auth.User) (*models.Dataproduct, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	querier := r.querier.WithTx(tx)

	dataproduct, err := querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:                  dp.Name,
		Description:           ptrToNullString(dp.Description),
		OwnerGroup:            dp.Group,
		OwnerTeamkatalogenUrl: ptrToNullString(dp.TeamkatalogenURL),
		Slug:                  slugify(dp.Slug, dp.Name),
		TeamContact:           ptrToNullString(dp.TeamContact),
		ProductAreaID:         ptrToNullString(dp.ProductAreaID),
		TeamID:                ptrToNullString(dp.TeamID),
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("rolling back dataproduct creation")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dataproductFromSQL(dataproduct), nil
}

func (r *Repo) UpdateDataproduct(ctx context.Context, id uuid.UUID, new models.UpdateDataproduct) (*models.Dataproduct, error) {
	res, err := r.querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:                  new.Name,
		Description:           ptrToNullString(new.Description),
		ID:                    id,
		OwnerTeamkatalogenUrl: ptrToNullString(new.TeamkatalogenURL),
		TeamContact:           ptrToNullString(new.TeamContact),
		Slug:                  slugify(new.Slug, new.Name),
		ProductAreaID:         ptrToNullString(new.ProductAreaID),
		TeamID:                ptrToNullString(new.TeamID),
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) DeleteDataproduct(ctx context.Context, id uuid.UUID) error {
	if err := r.querier.DeleteDataproduct(ctx, id); err != nil {
		return fmt.Errorf("deleting dataproduct from database: %w", err)
	}

	return nil
}

func (r *Repo) GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error) {
	return r.querier.GetBigqueryDatasources(ctx)
}

func (r *Repo) DataproductKeywords(ctx context.Context, prefix string) ([]*models.Keyword, error) {
	kws, err := r.querier.DataproductKeywords(ctx, prefix)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.Keyword, len(kws))
	for i, kw := range kws {
		ret[i] = &models.Keyword{
			Keyword: kw.Keyword,
			Count:   int(kw.Count),
		}
	}
	return ret, nil
}

func (r *Repo) DataproductGroupStats(ctx context.Context, limit, offset int) ([]*models.GroupStats, error) {
	stats, err := r.querier.DataproductGroupStats(ctx, gensql.DataproductGroupStatsParams{
		Lim:  int32(limit),
		Offs: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	ret := make([]*models.GroupStats, len(stats))
	for i, s := range stats {
		ret[i] = &models.GroupStats{
			Email:        s.Group,
			Dataproducts: int(s.Count),
		}
	}
	return ret, nil
}

func dataproductFromSQL(dp gensql.Dataproduct) *models.Dataproduct {
	return &models.Dataproduct{
		ID:           dp.ID,
		Name:         dp.Name,
		Created:      dp.Created,
		LastModified: dp.LastModified,
		Description:  nullStringToPtr(dp.Description),
		Slug:         dp.Slug,
		Owner: &models.Owner{
			Group:            dp.Group,
			TeamkatalogenURL: nullStringToPtr(dp.TeamkatalogenUrl),
			TeamContact:      nullStringToPtr(dp.TeamContact),
			ProductAreaID:    nullStringToPtr(dp.ProductAreaID),
			TeamID:           nullStringToPtr(dp.TeamID),
		},
	}
}
