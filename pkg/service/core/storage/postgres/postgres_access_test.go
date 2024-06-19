package postgres_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres/mock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAccessStorage_ListAccessRequestsForOwner(t *testing.T) {
	testCases := []struct {
		name           string
		owner          []string
		mockReturn     []gensql.DatasetAccessRequest
		mockReturnErr  error
		expectedResult []*service.AccessRequest
		expectedErr    error
	}{
		{
			name:  "Happy path",
			owner: []string{"owner1", "owner2"},
			mockReturn: []gensql.DatasetAccessRequest{
				{
					ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:   "type:subject1",
					Owner:     "owner1",
					Status:    "pending",
				},
			},
			mockReturnErr: nil,
			expectedResult: []*service.AccessRequest{
				{
					ID:          uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID:   uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:     "subject1",
					SubjectType: "type",
					Owner:       "owner1",
					Status:      "pending",
				},
			},
			expectedErr: nil,
		},
		{
			name:           "No access requests",
			owner:          []string{"owner1", "owner2"},
			mockReturn:     nil,
			mockReturnErr:  nil,
			expectedResult: nil,
			expectedErr:    nil,
		},
		{
			name:           "Error fetching access requests",
			owner:          []string{"owner1", "owner2"},
			mockReturn:     nil,
			mockReturnErr:  fmt.Errorf("error fetching access requests"),
			expectedResult: nil,
			expectedErr:    errs.E(errs.Database, errs.Op("postgres.ListAccessRequestsForOwner"), errs.Parameter("owner"), fmt.Errorf("error fetching access requests")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up the mock
			mockQueries := new(mock.AccessQueriesMock)
			mockTransacter := new(mock.MockTransacter)
			mockFn := mock.AccessQueriesWithTxFn(mockQueries, mockTransacter, nil)
			mockQueries.On("ListAccessRequestsForOwner", ctx, tc.owner).Return(tc.mockReturn, tc.mockReturnErr)

			// Call the method
			storage := postgres.NewAccessStorage(mockQueries, mockFn)
			result, err := storage.ListAccessRequestsForOwner(ctx, tc.owner)

			// Check the results
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedErr, err)
			mockQueries.AssertExpectations(t)
		})
	}
}
