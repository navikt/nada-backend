package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/navikt/nada-backend/pkg/database/gensql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreatePollyDocumentation(ctx context.Context, polly models.Polly) (models.DatabasePolly, error) {
	pollyDocumentation, err := r.querier.CreatePollyDocumentation(ctx, gensql.CreatePollyDocumentationParams{
		ExternalID: polly.ExternalID,
		Name:       polly.Name,
		Url:        polly.URL,
	})
	if err != nil {
		return models.DatabasePolly{}, err
	}

	return models.DatabasePolly{
		ID: pollyDocumentation.ID,
		Polly: models.Polly{
			ExternalID: pollyDocumentation.ExternalID,
			Name:       pollyDocumentation.Name,
			URL:        pollyDocumentation.Url,
		},
	}, nil
}

func (r *Repo) GetPollyDocumentation(ctx context.Context, accessID uuid.UUID) (*models.DatabasePolly, error) {
	pollySQL, err := r.querier.GetPollyDocumentation(ctx, accessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &models.DatabasePolly{
		ID: pollySQL.ID,
		Polly: models.Polly{
			ExternalID: pollySQL.ExternalID,
			Name:       pollySQL.Name,
			URL:        pollySQL.Url,
		},
	}, nil
}
