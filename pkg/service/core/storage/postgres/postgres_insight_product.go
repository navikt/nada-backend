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

var _ service.InsightProductStorage = &insightProductStorage{}

type insightProductStorage struct {
	db *database.Repo
}

func (s *insightProductStorage) DeleteInsightProduct(ctx context.Context, id uuid.UUID) error {
	err := s.db.Querier.DeleteInsightProduct(ctx, id)
	if err != nil {
		return errs.E(errs.Database, err)
	}

	return nil
}

func (s *insightProductStorage) CreateInsightProduct(ctx context.Context, creator string, input service.NewInsightProduct) (*service.InsightProduct, error) {
	insightProductSQL, err := s.db.Querier.CreateInsightProduct(ctx, gensql.CreateInsightProductParams{
		Name:             input.Name,
		Creator:          creator,
		Description:      ptrToNullString(input.Description),
		Keywords:         input.Keywords,
		OwnerGroup:       input.Group,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		Type:             input.Type,
		Link:             input.Link,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, err)
		}

		return nil, errs.E(errs.Database, err)
	}

	return s.GetInsightProductWithTeamkatalogen(ctx, insightProductSQL.ID)
}

func (s *insightProductStorage) UpdateInsightProduct(ctx context.Context, id uuid.UUID, input service.UpdateInsightProductDto) (*service.InsightProduct, error) {
	dbProduct, err := s.db.Querier.UpdateInsightProduct(ctx, gensql.UpdateInsightProductParams{
		ID:               id,
		Name:             input.Name,
		Description:      ptrToNullString(&input.Description),
		Keywords:         input.Keywords,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		Type:             input.TypeArg,
		Link:             input.Link,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, err)
		}

		return nil, errs.E(errs.Database, err)
	}

	return s.GetInsightProductWithTeamkatalogen(ctx, dbProduct.ID)
}

func (s *insightProductStorage) GetInsightProductWithTeamkatalogen(ctx context.Context, id uuid.UUID) (*service.InsightProduct, error) {
	raw, err := s.db.Querier.GetInsightProductWithTeamkatalogen(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, err)
		}

		return nil, err
	}

	return insightProductFromSQL(&raw), nil
}

func (s *insightProductStorage) GetInsightProductsByGroups(ctx context.Context, groups []string) ([]*service.InsightProduct, error) {
	raw, err := s.db.Querier.GetInsightProductsByGroups(ctx, groups)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errs.E(errs.Database, err)
	}

	insightProducts := make([]*service.InsightProduct, len(raw))
	for idx, ip := range raw {
		insightProducts[idx] = insightProductFromSQL(&ip)
	}

	return insightProducts, nil
}

func (s *insightProductStorage) GetInsightProductsByTeamID(ctx context.Context, teamIDs []string) ([]*service.InsightProduct, error) {
	raw, err := s.db.Querier.GetInsightProductsByProductArea(ctx, teamIDs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errs.E(errs.Database, err)
	}

	insightProducts := make([]*service.InsightProduct, len(raw))
	for idx, ip := range raw {
		insightProducts[idx] = insightProductFromSQL(&ip)
	}

	return insightProducts, nil
}

func (s *insightProductStorage) GetInsightProductsNumberByTeam(ctx context.Context, teamID string) (int64, error) {
	n, err := s.db.Querier.GetInsightProductsNumberByTeam(ctx, ptrToNullString(&teamID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, errs.E(errs.Database, err)
	}

	return n, nil
}

func insightProductFromSQL(insightProductSQL *gensql.InsightProductWithTeamkatalogenView) *service.InsightProduct {
	return &service.InsightProduct{
		ID:               insightProductSQL.ID,
		Name:             insightProductSQL.Name,
		Creator:          insightProductSQL.Creator,
		Created:          insightProductSQL.Created,
		Description:      insightProductSQL.Description.String,
		Type:             insightProductSQL.Type,
		Keywords:         insightProductSQL.Keywords,
		TeamkatalogenURL: nullStringToPtr(insightProductSQL.TeamkatalogenUrl),
		TeamID:           nullStringToPtr(insightProductSQL.TeamID),
		Group:            insightProductSQL.Group,
		Link:             insightProductSQL.Link,
		TeamName:         nullStringToPtr(insightProductSQL.TeamName),
		ProductAreaName:  nullStringToString(insightProductSQL.PaName),
	}
}

func NewInsightProductStorage(db *database.Repo) *insightProductStorage {
	return &insightProductStorage{
		db: db,
	}
}
