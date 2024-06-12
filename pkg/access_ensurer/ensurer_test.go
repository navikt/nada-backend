package access_ensurer

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/sirupsen/logrus"
)

var expired = []*service.Access{
	{ID: uuid.UUID{}},
	{ID: uuid.UUID{}},
	{ID: uuid.UUID{}},
}

func TestEnsurer(t *testing.T) {
	am := &MockAM{}
	mockServiceWrapper := &MockServiceWrapper{}
	bq := &MockBigQuery{}
	NewEnsurer(mockServiceWrapper, am, bq, nil, "", nil, logrus.StandardLogger().WithField("", "")).run(context.Background())

	if mockServiceWrapper.NGetUnrevokedExpiredAccess != 1 {
		t.Errorf("got: %v, want: %v", mockServiceWrapper.NGetUnrevokedExpiredAccess, 1)
	}
	if mockServiceWrapper.NGetBigqueryDatasource != len(expired) {
		t.Errorf("got: %v, want: %v", mockServiceWrapper.NGetBigqueryDatasource, len(expired))
	}
	if am.NRevoke != len(expired) {
		t.Errorf("got: %v, want: %v", am.NRevoke, len(expired))
	}
	if mockServiceWrapper.NRevokeAccessToDataset != len(expired) {
		t.Errorf("got: %v, want: %v", mockServiceWrapper.NRevokeAccessToDataset, len(expired))
	}
}

type MockServiceWrapper struct {
	NRevokeAccessToDataset         int
	NGetBigqueryDatasource         int
	NGetUnrevokedExpiredAccess     int
	NGetJoinableViewsWithReference int
	NListActiveAccessToDataset     int
	NGetOwnerGroupOfDataset        int
	NSetJoinableViewDeleted        int
}

func (m *MockServiceWrapper) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	m.NRevokeAccessToDataset++
	return nil
}

func (m *MockServiceWrapper) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID, isReference bool) (*service.BigQuery, *service.APIError) {
	m.NGetBigqueryDatasource++
	return &service.BigQuery{}, nil
}

func (m *MockServiceWrapper) GetUnrevokedExpiredAccess(ctx context.Context) ([]*service.Access, error) {
	m.NGetUnrevokedExpiredAccess++
	return expired, nil
}

func (m *MockServiceWrapper) GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error) {
	m.NGetJoinableViewsWithReference++
	return nil, nil
}

func (m *MockServiceWrapper) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]*service.Access, error) {
	m.NListActiveAccessToDataset++
	return nil, nil
}

func (m *MockServiceWrapper) GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	m.NGetOwnerGroupOfDataset++
	return "", nil
}

func (m *MockServiceWrapper) SetJoinableViewDeleted(ctx context.Context, joinableViewID uuid.UUID) error {
	m.NSetJoinableViewDeleted++
	return nil
}

func (m *MockServiceWrapper) GetPseudoDatasourcesToDelete(ctx context.Context) ([]*service.BigQuery, error) {
	return nil, nil
}

func (m *MockServiceWrapper) SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockServiceWrapper) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]gensql.GetJoinableViewsToBeDeletedWithRefDatasourceRow, error) {
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
