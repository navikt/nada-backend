package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/navikt/datakatalogen/backend/api"
	"github.com/navikt/datakatalogen/backend/config"
	"github.com/navikt/datakatalogen/backend/firestore"
	"github.com/navikt/datakatalogen/backend/iam"
	"github.com/stretchr/testify/assert"
)

type apiTestName struct {
	Name string
	Test apiTest
}

type apiTest struct {
	Method       string
	Path         string
	Body         string
	ExpectedCode int
	Headers      map[string]string
	After        func(t *testing.T, state map[string]string, body io.Reader)
}

var createDPloggedIn = apiTest{
	Method: "POST",
	Path:   "/api/v1/dataproducts",
	Body: `{
		"name": "cdp",
		"description": "desc",
		"datastore": [
			{
				"type": "` + iam.BucketType + `",
				"project_id": "project-id",
				"bucket_id": "the-bucket"
			}
		],
		"team": "team"
	}`,
	ExpectedCode: http.StatusCreated,
	After: func(t *testing.T, state map[string]string, body io.Reader) {
		id, _ := io.ReadAll(body)
		state["id"] = string(id)
	},
}

var apiTests = map[string][]apiTestName{
	"Update": {
		{
			Name: "Create dataproduct",
			Test: createDPloggedIn,
		},
		{
			Name: "Get dataproduct",
			Test: apiTest{
				Method:       "GET",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusOK,
				After: func(t *testing.T, state map[string]string, body io.Reader) {
					data := make(map[string]interface{})
					if err := json.NewDecoder(body).Decode(&data); err != nil {
						t.Error(err)
					}
					updated, _ := time.Parse(time.RFC3339, data["updated"].(string))
					state["updated"] = strconv.Itoa(int(updated.UnixNano()))
				},
			},
		},
		{
			Name: "Edit dataproduct",
			Test: apiTest{
				Method:       "PUT",
				Path:         "/api/v1/dataproducts/{id}",
				Body:         `{"name": "cdp2", "description": "desc", "datastore": [], "team": "team"}`,
				ExpectedCode: http.StatusOK,
			},
		},
		{
			Name: "Get updated dataproduct",
			Test: apiTest{
				Method:       "GET",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusOK,
				After: func(t *testing.T, state map[string]string, body io.Reader) {
					data := make(map[string]interface{})
					if err := json.NewDecoder(body).Decode(&data); err != nil {
						t.Error(err)
					}
					name := data["data_product"].(map[string]interface{})["name"]
					if name != "cdp2" {
						t.Errorf("Data product not updated correctly, expected 'cdp2', got '%v'", name)
					}

					updatedT, _ := time.Parse(time.RFC3339, data["updated"].(string))
					updated := int(updatedT.UnixNano())
					a, _ := strconv.Atoi(state["updated"])
					if updated <= a {
						t.Error("expected updated at to be different after update")
					}
				},
			},
		},
	},

	"Update different team": {
		{
			Name: "Create dataproduct",
			Test: createDPloggedIn,
		},
		{
			Name: "Edit dataproduct",
			Test: apiTest{
				Method:       "PUT",
				Path:         "/api/v1/dataproducts/{id}",
				Body:         `{"name": "cdp2", "description": "desc", "datastore": [], "team": "team"}`,
				Headers:      map[string]string{"X-Mock-Team": "otherteam"},
				ExpectedCode: http.StatusUnauthorized,
			},
		},
	},

	"Change access": {
		{
			Name: "Create dataproduct",
			Test: createDPloggedIn,
		},
		{
			Name: "Give access",
			Test: apiTest{
				Method:       "POST",
				Path:         "/api/v1/access/{id}",
				Body:         `{"subject": "me", "type": "user", "expires": "` + time.Now().Add(1*time.Hour).Format(time.RFC3339Nano) + `"}`,
				ExpectedCode: http.StatusOK,
			},
		},
		{
			Name: "Get updated dataproduct",
			Test: apiTest{
				Method:       "GET",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusOK,
				After: func(t *testing.T, state map[string]string, body io.Reader) {
					data := make(map[string]interface{})
					if err := json.NewDecoder(body).Decode(&data); err != nil {
						t.Error(err)
					}

					dp := data["data_product"].(map[string]interface{})
					_, ok := dp["access"].(map[string]interface{})["user:me"]
					if !ok {
						t.Error("expected to find access for 'user:me'")
					}
				},
			},
		},
		{
			Name: "Remove access",
			Test: apiTest{
				Method:       "DELETE",
				Path:         "/api/v1/access/{id}",
				Body:         `{"subject": "me", "type": "user"}`,
				ExpectedCode: http.StatusNoContent,
			},
		},
		{
			Name: "Get updated dataproduct after removal",
			Test: apiTest{
				Method:       "GET",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusOK,
				After: func(t *testing.T, state map[string]string, body io.Reader) {
					data := make(map[string]interface{})
					if err := json.NewDecoder(body).Decode(&data); err != nil {
						t.Error(err)
					}

					dp := data["data_product"].(map[string]interface{})
					_, ok := dp["access"].(map[string]interface{})["user:me"]
					if ok {
						t.Error("expected 'user:me' to no longer have access")
					}
				},
			},
		},
	},

	"Delete dataproduct": {
		{
			Name: "Create dataproduct",
			Test: createDPloggedIn,
		},
		{
			Name: "Delete the product",
			Test: apiTest{
				Method:       "DELETE",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusOK,
			},
		},
		{
			Name: "Try to get dataproduct",
			Test: apiTest{
				Method:       "GET",
				Path:         "/api/v1/dataproducts/{id}",
				ExpectedCode: http.StatusNotFound,
				After: func(t *testing.T, state map[string]string, body io.Reader) {
					io.Copy(os.Stdout, body)
				},
			},
		},
	},

	"Invalid datastore": {
		{
			Name: "Create dataproduct with invalid datastore",
			Test: apiTest{
				Method: "POST",
				Path:   "/api/v1/dataproducts",
				Body: `{
					"name": "cdp", 
					"description": "desc", 
					"datastore": [
						{
							"type": "` + iam.BucketType + `",
							"project_id": "invalid-id",
							"bucket_id": "the-bucket"
						}
					], 
					"team": "team"
				}`,
				ExpectedCode: http.StatusUnauthorized,
			},
		},
	},
}

func TestAPI(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	f, err := firestore.New(ctx, "aura-dev-api", "dp", "au")
	assert.NoError(t, err)

	mux := api.New(
		f,
		&mockAuthorizer{},
		config.Config{
			DevMode: true,
		},
		map[string]string{
			"team": "",
		},
		map[string][]string{
			"team": {"project-id"},
		},
	)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	client := testServer.Client()

	state := make(map[string]string)
	pathReplacer := regexp.MustCompile(`\{(\w+)\}`)

	for name, tcs := range apiTests {
		t.Run(name, func(t *testing.T) {
			for _, tc := range tcs {
				t.Run(tc.Name, func(t *testing.T) {
					var body io.Reader
					if tc.Test.Body != "" {
						body = strings.NewReader(tc.Test.Body)
					}

					// Allow `{var}` replacement in the path to fetch data from the state
					path := pathReplacer.ReplaceAllStringFunc(tc.Test.Path, func(s string) string {
						s = strings.Trim(s, "{}")
						return state[s]
					})

					req, err := http.NewRequest(tc.Test.Method, testServer.URL+path, body)
					if err != nil {
						t.Error(err)
					}

					for name, value := range tc.Test.Headers {
						req.Header.Add(name, value)
					}

					resp, err := client.Do(req)
					if err != nil {
						t.Error(err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != tc.Test.ExpectedCode {
						t.Errorf("expected %v, got %v (%v)", tc.Test.ExpectedCode, resp.StatusCode, http.StatusText(resp.StatusCode))
					}

					if tc.Test.After != nil {
						tc.Test.After(t, state, resp.Body)
					}
				})
			}
		})
	}
}

type mockAuthorizer struct{}

func (m *mockAuthorizer) RemoveDatastoreAccess(ctx context.Context, datastore map[string]string, subject string) error {
	return nil
}

func (m *mockAuthorizer) UpdateDatastoreAccess(ctx context.Context, datastore map[string]string, accessMap map[string]time.Time) error {
	return nil
}
