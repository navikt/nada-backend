package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.PollyStorage = &pollyStorage{}

type pollyStorage struct {
	db *database.Repo
}

func (s *pollyStorage) CreatePollyDocumentation(ctx context.Context, pollyInput service.PollyInput) (service.Polly, error) {
	const op errs.Op = "pollyStorage.CreatePollyDocumentation"

	pollyDocumentation, err := s.db.Querier.CreatePollyDocumentation(ctx, gensql.CreatePollyDocumentationParams{
		ExternalID: pollyInput.ExternalID,
		Name:       pollyInput.Name,
		Url:        pollyInput.URL,
	})
	if err != nil {
		return service.Polly{}, errs.E(errs.Database, op, err)
	}

	return service.Polly{
		ID: pollyDocumentation.ID,
		QueryPolly: service.QueryPolly{
			ExternalID: pollyDocumentation.ExternalID,
			Name:       pollyDocumentation.Name,
			URL:        pollyDocumentation.Url,
		},
	}, nil
}

func (s *pollyStorage) GetPollyDocumentation(ctx context.Context, id uuid.UUID) (*service.Polly, error) {
	const op errs.Op = "pollyStorage.GetPollyDocumentation"

	// TODO: either remove this or do it on database level for performance reasons
	pollyDoc, err := s.db.Querier.GetPollyDocumentation(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return &service.Polly{
		ID: pollyDoc.ID,
		QueryPolly: service.QueryPolly{
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
