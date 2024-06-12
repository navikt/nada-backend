package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/polly"
	"github.com/navikt/nada-backend/pkg/service"
)

type pollyStorage struct {
	db *database.Repo
}

func (s *pollyStorage) CreatePollyDocumentation(ctx context.Context, pollyInput service.PollyInput) (*service.Polly, error) {
	pollyDocumentation, err := s.db.Querier.CreatePollyDocumentation(ctx, gensql.CreatePollyDocumentationParams{
		ExternalID: pollyInput.ExternalID,
		Name:       pollyInput.Name,
		Url:        pollyInput.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("create polly documentation: %w", err)
	}

	return &service.Polly{
		ID: pollyDocumentation.ID,
		QueryPolly: polly.QueryPolly{
			ExternalID: pollyDocumentation.ExternalID,
			Name:       pollyDocumentation.Name,
			URL:        pollyDocumentation.Url,
		},
	}, nil
}

func (s *pollyStorage) GetPollyDocumentation(ctx context.Context, id uuid.UUID) (*service.Polly, error) {
	// TODO: either remove this or do it on database level for performance reasons
	pollyDoc, err := s.db.Querier.GetPollyDocumentation(ctx, id)
	if err != nil {
		return nil, err
	}

	return &service.Polly{
		ID: pollyDoc.ID,
		QueryPolly: polly.QueryPolly{
			ExternalID: pollyDoc.ExternalID,
			Name:       pollyDoc.Name,
			URL:        pollyDoc.Url,
		},
	}, nil
}

func NewPollyStorage(db *database.Repo) *pollyStorage {
	return &pollyStorage{
		db: db,
	}
}
