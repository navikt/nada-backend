package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetAccessDocumentation(ctx context.Context, accessID uuid.UUID) (*models.Polly, error) {
	pollySQL, err := r.querier.GetAccessDocumentation(ctx, accessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &models.Polly{
		ID:   pollySQL.PollyID,
		Name: pollySQL.PollyName,
		URL:  pollySQL.PollyUrl,
	}, nil
}
