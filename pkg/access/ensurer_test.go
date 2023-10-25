package access

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
)

var expired = []*models.Access{
	{ID: uuid.UUID{}},
	{ID: uuid.UUID{}},
	{ID: uuid.UUID{}},
}

func TestEnsurer(t *testing.T) {
	am := &MockAM{}
	repo := &MockRepo{}
	NewEnsurer(repo, am, nil, logrus.StandardLogger().WithField("", "")).run(context.Background())

	if repo.NGetUnrevokedExpiredAccess != 1 {
		t.Errorf("got: %v, want: %v", repo.NGetUnrevokedExpiredAccess, 1)
	}
	if repo.NGetBigqueryDatasource != len(expired) {
		t.Errorf("got: %v, want: %v", repo.NGetBigqueryDatasource, len(expired))
	}
	if am.NRevoke != len(expired) {
		t.Errorf("got: %v, want: %v", am.NRevoke, len(expired))
	}
	if repo.NRevokeAccessToDataset != len(expired) {
		t.Errorf("got: %v, want: %v", repo.NRevokeAccessToDataset, len(expired))
	}
}

type MockRepo struct {
	NRevokeAccessToDataset     int
	NGetBigqueryDatasource     int
	NGetUnrevokedExpiredAccess int
}

func (m *MockRepo) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	m.NRevokeAccessToDataset++
	return nil
}

func (m *MockRepo) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (models.BigQuery, error) {
	m.NGetBigqueryDatasource++
	return models.BigQuery{}, nil
}

func (m *MockRepo) GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error) {
	m.NGetUnrevokedExpiredAccess++
	return expired, nil
}

type MockAM struct {
	NRevoke int
}

func (a *MockAM) Revoke(ctx context.Context, projectID, dataset, table, member string) error {
	a.NRevoke++
	return nil
}
