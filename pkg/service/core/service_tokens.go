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

func (s *tokenService) RotateNadaToken(ctx context.Context, team string) error {
	const op errs.Op = "tokenService.RotateNadaToken"

	if team == "" {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("no team provided"))
	}

	if err := ensureUserInGroup(ctx, team+"@nav.no"); err != nil {
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
