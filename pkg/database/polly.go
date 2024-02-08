package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/navikt/nada-backend/pkg/database/gensql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreatePollyDocumentation(ctx context.Context, polly models.PollyInput) (models.Polly, error) {
	pollyDocumentation, err := r.Querier.CreatePollyDocumentation(ctx, gensql.CreatePollyDocumentationParams{
		ExternalID: polly.ExternalID,
		Name:       polly.Name,
		Url:        polly.URL,
	})
	if err != nil {
		return models.Polly{}, err
	}

	return models.Polly{
		ID: pollyDocumentation.ID,
		QueryPolly: models.QueryPolly{
			ExternalID: pollyDocumentation.ExternalID,
			Name:       pollyDocumentation.Name,
			URL:        pollyDocumentation.Url,
		},
	}, nil
}

func (r *Repo) GetPollyDocumentation(ctx context.Context, accessID uuid.UUID) (*models.Polly, error) {
	pollySQL, err := r.Querier.GetPollyDocumentation(ctx, accessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &models.Polly{
		ID: pollySQL.ID,
		QueryPolly: models.QueryPolly{
			ExternalID: pollySQL.ExternalID,
			Name:       pollySQL.Name,
			URL:        pollySQL.Url,
		},
	}, nil
}
