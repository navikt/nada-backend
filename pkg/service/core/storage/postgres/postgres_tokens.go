package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TokenStorage = &tokenStorage{}

type tokenStorage struct {
	db *database.Repo
}

func (s *tokenStorage) GetNadaToken(ctx context.Context, team string) (string, error) {
	const op errs.Op = "postgres.GetNadaToken"

	token, err := s.db.Querier.GetNadaToken(ctx, team)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errs.E(errs.NotExist, op, err)
		}

		return "", errs.E(errs.Database, op, err)
	}

	return token.String(), nil
}

func (s *tokenStorage) RotateNadaToken(ctx context.Context, team string) error {
	const op errs.Op = "postgres.RotateNadaToken"

	err := s.db.Querier.RotateNadaToken(ctx, team)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *tokenStorage) GetNadaTokensForTeams(ctx context.Context, teams []string) ([]service.NadaToken, error) {
	const op errs.Op = "postgres.GetNadaTokensForTeams"

	rawTokens, err := s.db.Querier.GetNadaTokensForTeams(ctx, teams)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	tokens := make([]service.NadaToken, len(rawTokens))
	for i, t := range rawTokens {
		tokens[i] = service.NadaToken{
			Team:  t.Team,
			Token: t.Token,
		}
	}

	return tokens, nil
}

func (s *tokenStorage) GetNadaTokens(ctx context.Context) (map[string]string, error) {
	const op errs.Op = "postgres.GetNadaTokens"

	rawTokens, err := s.db.Querier.GetNadaTokens(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	tokens := map[string]string{}
	for _, t := range rawTokens {
		tokens[t.Token.String()] = t.Team
	}

	return tokens, nil
}

func NewTokenStorage(db *database.Repo) *tokenStorage {
	return &tokenStorage{
		db: db,
	}
}
