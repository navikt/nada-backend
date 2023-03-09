package database

import (
	"context"

	"github.com/google/uuid"
)

func (r *Repo) GetNadaToken(ctx context.Context, team string) (uuid.UUID, error) {
	return r.querier.GetNadaToken(ctx, team)
}

func (r *Repo) DeleteNadaToken(ctx context.Context, team string) error {
	return r.querier.DeleteNadaToken(ctx, team)
}
