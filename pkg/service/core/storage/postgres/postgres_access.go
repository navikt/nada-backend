package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
)

type accessStorage struct {
	db *database.Repo
}

func (s *accessStorage) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error) {
	access, err := s.db.Querier.ListActiveAccessToDataset(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("list active access to dataset: %w", err)
	}

	var ret []*service.Access
	for _, e := range access {
		ret = append(ret, accessFromSQL(e))
	}

	return ret, nil
}

func accessFromSQL(access gensql.DatasetAccess) *service.Access {
	return &service.Access{
		ID:              access.ID,
		Subject:         access.Subject,
		Granter:         access.Granter,
		Expires:         nullTimeToPtr(access.Expires),
		Created:         access.Created,
		Revoked:         nullTimeToPtr(access.Revoked),
		DatasetID:       access.DatasetID,
		AccessRequestID: nullUUIDToUUIDPtr(access.AccessRequestID),
	}
}

func NewAccessStorage(db *database.Repo) *accessStorage {
	return &accessStorage{
		db: db,
	}
}
