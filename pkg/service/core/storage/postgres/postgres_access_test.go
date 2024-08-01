package postgres_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/stretchr/testify/assert"
)

func TestDatasetAccessRequest_To(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          gensql.DatasetAccessRequest
		expectedResult *service.AccessRequest
		expectedErr    error
	}{
		{
			name: "Happy path",
			input: gensql.DatasetAccessRequest{
				ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				Subject:   "user:subject1",
				Owner:     "owner1",
				Status:    "pending",
			},
			expectedResult: &service.AccessRequest{
				ID:          uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				DatasetID:   uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				Subject:     "subject1",
				SubjectType: "user",
				Owner:       "owner1",
				Status:      "pending",
			},
			expectedErr: nil,
		},
		{
			name: "Error parsing subject",
			input: gensql.DatasetAccessRequest{
				ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				Subject:   "subject1",
				Owner:     "owner1",
				Status:    "pending",
			},
			expectedResult: nil,
			expectedErr:    errs.E(errs.Internal, errs.Op("DatasetAccessRequest.To"), fmt.Errorf("subject1 is not a valid subject, expected [subject_type:subject]")),
		},
		{
			name: "Invalid subject type",
			input: gensql.DatasetAccessRequest{
				ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
				Subject:   "invalid_type:subject1",
				Owner:     "owner1",
				Status:    "pending",
			},
			expectedResult: nil,
			expectedErr:    errs.E(errs.Internal, errs.Op("DatasetAccessRequest.To"), fmt.Errorf("invalid_type is not a valid subject type, expected one of [service_account, user, group]")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := postgres.DatasetAccessRequest(tc.input).To()

			assert.Equal(t, tc.expectedResult, got)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestDatasetAccessRequests_To(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          []gensql.DatasetAccessRequest
		expectedResult []*service.AccessRequest
		expectedErr    error
	}{
		{
			name: "Happy path",
			input: []gensql.DatasetAccessRequest{
				{
					ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:   "user:subject1",
					Owner:     "owner1",
					Status:    "pending",
				},
				{
					ID:        uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID: uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:   "user:subject2",
					Owner:     "owner2",
					Status:    "pending",
				},
			},
			expectedResult: []*service.AccessRequest{
				{
					ID:          uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID:   uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:     "subject1",
					SubjectType: "user",
					Owner:       "owner1",
					Status:      "pending",
				},
				{
					ID:          uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					DatasetID:   uuid.MustParse("14726B25-FACE-47C7-AC55-782799362E58"),
					Subject:     "subject2",
					SubjectType: "user",
					Owner:       "owner2",
					Status:      "pending",
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := postgres.DatasetAccessRequests(tc.input).To()
			assert.Equal(t, tc.expectedResult, got)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
