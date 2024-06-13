package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.InsightProductStorage = &insightProductStorage{}

type insightProductStorage struct {
	db *database.Repo
}

func (s *insightProductStorage) GetInsightProductsByGroups(ctx context.Context, groups []string) ([]*service.InsightProduct, error) {
	raw, err := s.db.Querier.GetInsightProductsByGroups(ctx, groups)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	insightProducts := make([]*service.InsightProduct, len(raw))
	for idx, ip := range raw {
		insightProducts[idx] = insightProductFromSQL(&ip)
	}

	return insightProducts, nil
}

func (s *insightProductStorage) GetInsightProductsByTeamID(ctx context.Context, teamIDs []string) ([]*service.InsightProduct, error) {
	raw, err := s.db.Querier.GetInsightProductsByProductArea(ctx, teamIDs)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
		return 0, fmt.Errorf("failed to get insight products number: %w", err)
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
