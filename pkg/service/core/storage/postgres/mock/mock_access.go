package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/stretchr/testify/mock"
)

var _ postgres.AccessQueries = &AccessQueriesMock{}

type AccessQueriesMock struct {
	mock.Mock
}

func AccessQueriesWithTxFn(m *AccessQueriesMock, t database.Transacter, err error) func() (postgres.AccessQueries, database.Transacter, error) {
	return func() (postgres.AccessQueries, database.Transacter, error) {
		return m, t, err
	}
}

func (m *AccessQueriesMock) ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]gensql.DatasetAccessRequest, error) {
	args := m.Called(ctx, owner)
	return args.Get(0).([]gensql.DatasetAccessRequest), args.Error(1)
}

func (m *AccessQueriesMock) ListUnrevokedExpiredAccessEntries(ctx context.Context) ([]gensql.DatasetAccess, error) {
	args := m.Called(ctx)
	return args.Get(0).([]gensql.DatasetAccess), args.Error(1)
}

func (m *AccessQueriesMock) ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]gensql.DatasetAccess, error) {
	args := m.Called(ctx, datasetID)
	return args.Get(0).([]gensql.DatasetAccess), args.Error(1)
}

func (m *AccessQueriesMock) ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]gensql.DatasetAccessRequest, error) {
	args := m.Called(ctx, datasetID)
	return args.Get(0).([]gensql.DatasetAccessRequest), args.Error(1)
}

func (m *AccessQueriesMock) CreateAccessRequestForDataset(ctx context.Context, params gensql.CreateAccessRequestForDatasetParams) (gensql.DatasetAccessRequest, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(gensql.DatasetAccessRequest), args.Error(1)
}

func (m *AccessQueriesMock) GetAccessRequest(ctx context.Context, id uuid.UUID) (gensql.DatasetAccessRequest, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(gensql.DatasetAccessRequest), args.Error(1)
}

func (m *AccessQueriesMock) DeleteAccessRequest(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *AccessQueriesMock) UpdateAccessRequest(ctx context.Context, params gensql.UpdateAccessRequestParams) (gensql.DatasetAccessRequest, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(gensql.DatasetAccessRequest), args.Error(1)
}

func (m *AccessQueriesMock) GrantAccessToDataset(ctx context.Context, params gensql.GrantAccessToDatasetParams) (gensql.DatasetAccess, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(gensql.DatasetAccess), args.Error(1)
}

func (m *AccessQueriesMock) ApproveAccessRequest(ctx context.Context, params gensql.ApproveAccessRequestParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *AccessQueriesMock) GetActiveAccessToDatasetForSubject(ctx context.Context, params gensql.GetActiveAccessToDatasetForSubjectParams) (gensql.DatasetAccess, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(gensql.DatasetAccess), args.Error(1)
}

func (m *AccessQueriesMock) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *AccessQueriesMock) DenyAccessRequest(ctx context.Context, params gensql.DenyAccessRequestParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *AccessQueriesMock) GetAccessToDataset(ctx context.Context, id uuid.UUID) (gensql.DatasetAccess, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(gensql.DatasetAccess), args.Error(1)
}
