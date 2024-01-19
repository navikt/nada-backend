package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetNadaToken(ctx context.Context, team string) (uuid.UUID, error) {
	return r.Querier.GetNadaToken(ctx, team)
}

func (r *Repo) GetNadaTokens(ctx context.Context, teams []string) ([]*models.NadaToken, error) {
	tokensSQL, err := r.Querier.GetNadaTokens(ctx, teams)
	if err != nil {
		return nil, err
	}

	tokens := make([]*models.NadaToken, len(tokensSQL))
	for i, t := range tokensSQL {
		tokens[i] = &models.NadaToken{
			Team:  t.Team,
			Token: t.Token,
		}
	}
	return tokens, nil
}

func (r *Repo) DeleteNadaToken(ctx context.Context, team string) error {
	return r.Querier.DeleteNadaToken(ctx, team)
}

func (r *Repo) GetTeamFromToken(ctx context.Context, token uuid.UUID) (string, error) {
	return r.Querier.GetTeamFromNadaToken(ctx, token)
}
