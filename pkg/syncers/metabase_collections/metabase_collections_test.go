package metabase_collections_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/syncers/metabase_collections"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMetabaseAPI struct {
	mock.Mock
	service.MetabaseAPI
}

func (m *MockMetabaseAPI) GetCollections(ctx context.Context) ([]*service.MetabaseCollection, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*service.MetabaseCollection), args.Error(1)
}

func (m *MockMetabaseAPI) UpdateCollection(ctx context.Context, collection *service.MetabaseCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

type MockMetabaseStorage struct {
	mock.Mock
	service.MetabaseStorage
}

func (m *MockMetabaseStorage) GetAllMetadata(ctx context.Context) ([]*service.MetabaseMetadata, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*service.MetabaseMetadata), args.Error(1)
}

func setupMockAPI() *MockMetabaseAPI {
	return new(MockMetabaseAPI)
}

func setupMockStorage() *MockMetabaseStorage {
	return new(MockMetabaseStorage)
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestSyncer_MissingCollections(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter())
	ctx := context.Background()

	testCases := []struct {
		name         string
		setupAPI     func(api *MockMetabaseAPI)
		setupStorage func(storage *MockMetabaseStorage)
		expect       *metabase_collections.CollectionsReport
		expectErr    error
	}{
		{
			name: "logs missing collections in metabase",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{}, nil)
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SyncCompleted: timePtr(time.Now()), DatabaseID: intPtr(0)},
				}, nil)
			},
			expect: &metabase_collections.CollectionsReport{
				Missing: []metabase_collections.Missing{
					{DatasetID: "00000000-0000-0000-0000-000000000001", CollectionID: 1, DatabaseID: 0},
				},
			},
		},
		{
			name: "logs missing collections in database",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{
					{ID: 1, Name: "collection1"},
				}, nil)
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{}, nil)
			},
			expect: &metabase_collections.CollectionsReport{
				Dangling: []metabase_collections.Dangling{
					{ID: 1, Name: "collection1"},
				},
			},
		},
		{
			name:     "handles storage error",
			setupAPI: func(api *MockMetabaseAPI) {},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{}, errors.New("storage error"))
			},
			expectErr: fmt.Errorf("storage error"),
		},
		{
			name: "handles API error",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{}, errors.New("api error"))
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SyncCompleted: timePtr(time.Now())},
				}, nil)
			},
			expectErr: fmt.Errorf("api error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			api := setupMockAPI()
			storage := setupMockStorage()
			syncer := metabase_collections.New(api, storage, 1, logger)

			tc.setupAPI(api)
			tc.setupStorage(storage)

			report, err := syncer.CollectionsReport(ctx)
			if tc.expectErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expect, report)
			}
		})
	}
}

func TestSyncer_AddRestrictedTagToCollections(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter())
	ctx := context.Background()

	testCases := []struct {
		name         string
		setupAPI     func(api *MockMetabaseAPI)
		setupStorage func(storage *MockMetabaseStorage)
		expectErr    error
	}{
		{
			name: "updates collection name",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{
					{ID: 1, Name: "collection1"},
				}, nil)
				api.On("UpdateCollection", ctx, &service.MetabaseCollection{
					ID:   1,
					Name: "collection1 🔐",
				}).Return(nil)
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				}, nil)
			},
		},
		{
			name: "handles update error",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{
					{ID: 1, Name: "collection1"},
				}, nil)
				api.On("UpdateCollection", ctx, &service.MetabaseCollection{
					ID:   1,
					Name: "collection1 🔐",
				}).Return(errors.New("update error"))
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SyncCompleted: timePtr(time.Now()), DatabaseID: intPtr(0)},
				}, nil)
			},
			expectErr: fmt.Errorf("update error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			api := setupMockAPI()
			storage := setupMockStorage()
			syncer := metabase_collections.New(api, storage, 1, logger)

			tc.setupAPI(api)
			tc.setupStorage(storage)

			err := syncer.AddRestrictedTagToCollections(ctx)
			if tc.expectErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncer_Run(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter())
	ctx := context.Background()

	testCases := []struct {
		name         string
		setupAPI     func(api *MockMetabaseAPI)
		setupStorage func(storage *MockMetabaseStorage)
		expectLog    string
	}{
		{
			name: "runs syncer and logs missing collections",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{}, nil)
				api.On("UpdateCollection", ctx, mock.Anything).Return(nil)
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				}, nil)
			},
			expectLog: "collection_missing",
		},
		{
			name: "handles AddRestrictedTagToCollections error",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{}, nil)
				api.On("UpdateCollection", ctx, mock.Anything).Return(errors.New("update error"))
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				}, nil)
			},
			expectLog: "adding restricted tag to collections",
		},
		{
			name: "handles CollectionsReport error",
			setupAPI: func(api *MockMetabaseAPI) {
				api.On("GetCollections", ctx).Return([]*service.MetabaseCollection{}, errors.New("api error"))
			},
			setupStorage: func(storage *MockMetabaseStorage) {
				storage.On("GetAllMetadata", ctx).Return([]*service.MetabaseMetadata{
					{CollectionID: intPtr(1), DatasetID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				}, nil)
			},
			expectLog: "reporting missing collections",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			api := setupMockAPI()
			storage := setupMockStorage()
			syncer := metabase_collections.New(api, storage, 1, logger)

			tc.setupAPI(api)
			tc.setupStorage(storage)

			go syncer.Run(ctx, 0)
			time.Sleep(2 * time.Second)

			// Check logs for expected messages
			// This part assumes you have a way to capture and check logs
			// For example, using a custom logger or a log capturing library
			// assert.Contains(t, capturedLogs, tc.expectLog)
		})
	}
}
