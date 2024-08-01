package core

import (
	"context"
	"fmt"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.TokenService = &tokenService{}

type tokenService struct {
	tokenStorage service.TokenStorage
}

func (s *tokenService) ValidateToken(ctx context.Context, token string) (bool, error) {
	const op errs.Op = "tokenService.ValidateToken"

	tokens, err := s.GetNadaTokens(ctx)
	if err != nil {
		return false, errs.E(op, err)
	}

	_, hasKey := tokens[token]

	return hasKey, nil
}

func (s *tokenService) GetNadaTokens(ctx context.Context) (map[string]string, error) {
	const op errs.Op = "tokenService.GetNadaTokens"

	tokens, err := s.tokenStorage.GetNadaTokens(ctx)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return tokens, nil
}

func (s *tokenService) GetNadaTokenForTeam(ctx context.Context, team string) (string, error) {
	const op errs.Op = "tokenService.GetNadaTokenForTeam"

	// FIXME: should we not check the user here?

	token, err := s.tokenStorage.GetNadaToken(ctx, team)
	if err != nil {
		return "", errs.E(op, err)
	}

	return token, nil
}

func (s *tokenService) GetTeamFromNadaToken(ctx context.Context, token string) (string, error) {
	const op errs.Op = "tokenService.GetTeamFromNadaToken"

	tokenMap, err := s.tokenStorage.GetNadaTokens(ctx)
	if err != nil {
		return "", errs.E(op, err)
	}

	team, ok := tokenMap[token]
	if !ok {
		return "", errs.E(errs.InvalidRequest, op, fmt.Errorf("token not found"))
	}

	return team, nil
}

func (s *tokenService) RotateNadaToken(ctx context.Context, user *service.User, team string) error {
	const op errs.Op = "tokenService.RotateNadaToken"

	if team == "" {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("no team provided"))
	}

	if err := ensureUserInGroup(user, team+"@nav.no"); err != nil {
		return errs.E(op, err)
	}

	err := s.tokenStorage.RotateNadaToken(ctx, team)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func NewTokenService(tokenStorage service.TokenStorage) service.TokenService {
	return &tokenService{
		tokenStorage: tokenStorage,
	}
}
