package tk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/stretchr/testify/assert"
)

func TestClient_GetProductAreas(t *testing.T) {
	testCases := []struct {
		name      string
		body      *tk.ProductAreas
		err       error
		expectErr bool
	}{
		{
			name: "should return product areas",
			body: &tk.ProductAreas{
				Content: []tk.ProductArea{
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Name: "Product Area 1",
					},
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
						Name: "Product Area 2",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "should return error",
			err:       assert.AnError,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Header.Get(tk.ConsumerIDHeader), tk.ConsumerID)
				assert.Equal(t, r.URL.Path, "/productarea")
				assert.Equal(t, r.Method, http.MethodGet)
				assert.Equal(t, r.URL.Query().Get("status"), "ACTIVE")

				if tc.err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tc.body)
				assert.NoError(t, err)
			}))

			client := tk.New(testServer.URL, http.DefaultClient)
			got, err := client.GetProductAreas(context.Background())
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.body, got)
			}
		})
	}
}

func TestClient_GetTeam(t *testing.T) {
	testCases := []struct {
		name      string
		teamID    uuid.UUID
		body      *tk.Team
		err       error
		expectErr bool
	}{
		{
			name:   "should return team",
			teamID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			body: &tk.Team{
				ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Name: "Team 1",
			},
			expectErr: false,
		},
		{
			name:      "should return error",
			teamID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			err:       assert.AnError,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Header.Get(tk.ConsumerIDHeader), tk.ConsumerID)
				assert.Equal(t, r.URL.Path, "/team/"+tc.teamID.String())
				assert.Equal(t, r.Method, http.MethodGet)

				if tc.err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tc.body)
				assert.NoError(t, err)
			}))

			client := tk.New(testServer.URL, http.DefaultClient)
			got, err := client.GetTeam(context.Background(), tc.teamID)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.body, got)
			}
		})
	}
}

func TestClient_GetTeamsInProductArea(t *testing.T) {
	testCases := []struct {
		name          string
		productAreaID uuid.UUID
		body          *tk.Teams
		err           error
		expectErr     bool
	}{
		{
			name:          "should return teams",
			productAreaID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			body: &tk.Teams{
				Content: []tk.Team{
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Name: "Team 1",
					},
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
						Name: "Team 2",
					},
				},
			},
			expectErr: false,
		},
		{
			name:          "should return error",
			productAreaID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			err:           assert.AnError,
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Header.Get(tk.ConsumerIDHeader), tk.ConsumerID)
				assert.Equal(t, r.URL.Path, "/team")
				assert.Equal(t, r.URL.Query().Get("productAreaId"), tc.productAreaID.String())
				assert.Equal(t, r.URL.Query().Get("status"), "ACTIVE")
				assert.Equal(t, r.Method, http.MethodGet)

				if tc.err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tc.body)
				assert.NoError(t, err)
			}))

			client := tk.New(testServer.URL, http.DefaultClient)
			got, err := client.GetTeamsInProductArea(context.Background(), tc.productAreaID)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.body, got)
			}
		})
	}
}

func TestClient_GetTeams(t *testing.T) {
	testCases := []struct {
		name      string
		body      *tk.Teams
		err       error
		expectErr bool
	}{
		{
			name: "should return teams",
			body: &tk.Teams{
				Content: []tk.Team{
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Name: "Team 1",
					},
					{
						ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
						Name: "Team 2",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "should return error",
			err:       assert.AnError,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Header.Get(tk.ConsumerIDHeader), tk.ConsumerID)
				assert.Equal(t, r.URL.Path, "/team")
				assert.Equal(t, r.URL.Query().Get("status"), "ACTIVE")
				assert.Equal(t, r.Method, http.MethodGet)

				if tc.err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(tc.body)
				assert.NoError(t, err)
			}))

			client := tk.New(testServer.URL, http.DefaultClient)
			got, err := client.GetTeams(context.Background())
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.body, got)
			}
		})
	}
}
