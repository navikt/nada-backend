package service

import (
	"context"

	"github.com/google/uuid"
)

type TokenStorage interface {
	GetNadaTokensForTeams(ctx context.Context, teams []string) ([]NadaToken, error)
	GetNadaTokens(ctx context.Context) (map[string]string, error)
	GetNadaToken(ctx context.Context, team string) (string, error)
	RotateNadaToken(ctx context.Context, team string) error
}

type TokenService interface {
	RotateNadaToken(ctx context.Context, user *User, team string) error
	GetTeamFromNadaToken(ctx context.Context, token string) (string, error)
	GetNadaTokenForTeam(ctx context.Context, team string) (string, error)
	GetNadaTokens(ctx context.Context) (map[string]string, error)
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type NadaToken struct {
	Team  string    `json:"team"`
	Token uuid.UUID `json:"token"`
}
