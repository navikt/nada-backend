package access

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
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
	bq := &MockBigQuery{}
	NewEnsurer(repo, am, bq, nil, "", nil, logrus.StandardLogger().WithField("", "")).run(context.Background())

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
	NRevokeAccessToDataset         int
	NGetBigqueryDatasource         int
	NGetUnrevokedExpiredAccess     int
	NGetJoinableViewsWithReference int
	NListActiveAccessToDataset     int
	NGetOwnerGroupOfDataset        int
	NSetJoinableViewDeleted        int
}

func (m *MockRepo) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	m.NRevokeAccessToDataset++
	return nil
}

func (m *MockRepo) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID, isReference bool) (models.BigQuery, error) {
	m.NGetBigqueryDatasource++
	return models.BigQuery{}, nil
}

func (m *MockRepo) GetUnrevokedExpiredAccess(ctx context.Context) ([]*models.Access, error) {
	m.NGetUnrevokedExpiredAccess++
	return expired, nil
}

func (m *MockRepo) GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error) {
	m.NGetJoinableViewsWithReference++
	return nil, nil
}

func (m *MockRepo) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*models.Access, error) {
	m.NListActiveAccessToDataset++
	return nil, nil
}

func (m *MockRepo) GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	m.NGetOwnerGroupOfDataset++
	return "", nil
}

func (m *MockRepo) SetJoinableViewDeleted(ctx context.Context, joinableViewID uuid.UUID) error {
	m.NSetJoinableViewDeleted++
	return nil
}

func (m *MockRepo) GetPseudoDatasourcesToDelete(ctx context.Context) ([]*models.BigQuery, error) {
	return nil, nil
}

func (m *MockRepo) SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockRepo) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]gensql.GetJoinableViewsToBeDeletedWithRefDatasourceRow, error) {
	return nil, nil
}

type MockAM struct {
	NGrant  int
	NRevoke int
}

func (a *MockAM) Grant(ctx context.Context, projectID, dataset, table, member string) error {
	a.NGrant++
	return nil
}

func (a *MockAM) Revoke(ctx context.Context, projectID, dataset, table, member string) error {
	a.NRevoke++
	return nil
}

type MockBigQuery struct {
	NDeleteDataset int
}

func (b *MockBigQuery) DeleteJoinableDataset(ctx context.Context, datasetID string) error {
	b.NDeleteDataset++
	return nil
}

func (b *MockBigQuery) DeleteJoinableView(ctx context.Context, joinableViewName, refProjectID, refDatasetID, refTableID string) error {
	return nil
}

func (b *MockBigQuery) DeletePseudoView(ctx context.Context, pseudoProjectID, pseudoDatasetID, pseudoTableID string) error {
	return nil
}
