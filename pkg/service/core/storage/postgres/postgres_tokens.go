package postgres

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TokenStorage = &tokenStorage{}

type tokenStorage struct {
	db *database.Repo
}

func (s *tokenStorage) GetNadaToken(ctx context.Context, team string) (string, error) {
	token, err := s.db.Querier.GetNadaToken(ctx, team)
	if err != nil {
		return "", fmt.Errorf("failed to get nada token: %w", err)
	}

	return token.String(), nil
}

func (s *tokenStorage) RotateNadaToken(ctx context.Context, team string) error {
	err := s.db.Querier.RotateNadaToken(ctx, team)
	if err != nil {
		return fmt.Errorf("failed to rotate nada token: %w", err)
	}

	return nil
}

func (s *tokenStorage) GetNadaTokensForTeams(ctx context.Context, teams []string) ([]service.NadaToken, error) {
	rawTokens, err := s.db.Querier.GetNadaTokensForTeams(ctx, teams)
	if err != nil {
		return nil, err
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
	rawTokens, err := s.db.Querier.GetNadaTokens(ctx)
	if err != nil {
		return nil, err
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
