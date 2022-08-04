package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateQuarto(ctx context.Context, team, data string) (uuid.UUID, error) {
	quartoSQL, err := r.querier.CreateQuarto(ctx, gensql.CreateQuartoParams{
		Team:    team,
		Content: data,
	})

	return quartoSQL.ID, err
}

func (r *Repo) GetQuarto(ctx context.Context, id uuid.UUID) (*models.Quarto, error) {
	quartoSQL, err := r.querier.GetQuarto(ctx, id)

	return quartoSQLToGraphql(quartoSQL), err
}

func (r *Repo) GetQuartos(ctx context.Context) ([]*models.Quarto, error) {
	quartoSQLs, err := r.querier.GetQuartos(ctx)
	if err != nil {
		return nil, err
	}

	quartoGraphqls := make([]*models.Quarto, len(quartoSQLs))
	for idx, quarto := range quartoSQLs {
		quartoGraphqls[idx] = quartoSQLToGraphql(quarto)
	}

	return quartoGraphqls, err
}

func quartoSQLToGraphql(quarto gensql.Quarto) *models.Quarto {
	return &models.Quarto{
		ID: quarto.ID,
		Team: &models.Owner{
			Group:            quarto.Team,
			TeamkatalogenURL: stringToPtr(""),
		},
		Content: quarto.Content,
	}
}
