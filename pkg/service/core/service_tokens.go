package core

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TokenService = &tokenService{}

type tokenService struct {
	tokenStorage service.TokenStorage
}

func (s *tokenService) GetNadaTokenForTeam(ctx context.Context, team string) (string, error) {
	token, err := s.tokenStorage.GetNadaToken(ctx, team)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	return token, nil
}

func (s *tokenService) GetTeamFromNadaToken(ctx context.Context, token string) (string, error) {
	tokenMap, err := s.tokenStorage.GetNadaTokens(ctx)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	team, ok := tokenMap[token]
	if !ok {
		return "", fmt.Errorf("token not found")
	}

	return team, nil
}

func (s *tokenService) RotateNadaToken(ctx context.Context, team string) error {
	if team == "" {
		return fmt.Errorf("no team provided")
	}

	if err := ensureUserInGroup(ctx, team+"@nav.no"); err != nil {
		return fmt.Errorf("user not in gcp group: %w", err)
	}

	err := s.tokenStorage.RotateNadaToken(ctx, team)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}

func NewTokenService(tokenStorage service.TokenStorage) service.TokenService {
	return &tokenService{
		tokenStorage: tokenStorage,
	}
}
