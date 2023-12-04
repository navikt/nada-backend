package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetNadaToken(ctx context.Context, team string) (uuid.UUID, error) {
	return r.querier.GetNadaToken(ctx, team)
}

func (r *Repo) GetNadaTokens(ctx context.Context, teams []string) ([]*models.NadaToken, error) {
	tokensSQL, err := r.querier.GetNadaTokens(ctx, teams)
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
	return r.querier.DeleteNadaToken(ctx, team)
}

func (r *Repo) GetTeamFromToken(ctx context.Context, token uuid.UUID) (string, error) {
	return r.querier.GetTeamFromNadaToken(ctx, token)
}

func (r *Repo) CreateTeamProductAreaMapping(ctx context.Context, tx *sql.Tx, teamID, productAreaID *string) error {
	querier := r.querier.WithTx(tx)
	if teamID := ptrToString(teamID); teamID != "" {
		_, err := querier.GetTeamAndProductAreaID(ctx, teamID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				if err := tx.Rollback(); err != nil {
					r.log.WithError(err).Error("rolling back story create, get team and product area id")
				}
				return err
			}

			_, err = querier.CreateTeamAndProductAreaMapping(ctx, gensql.CreateTeamAndProductAreaMappingParams{
				TeamID:        teamID,
				ProductAreaID: ptrToNullString(productAreaID),
			})
			if err != nil {
				if err := tx.Rollback(); err != nil {
					r.log.WithError(err).Error("rolling back story create, insert team and product mapping")
				}
				return err
			}
		}
	}

	return nil
}
