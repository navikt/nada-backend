//go:build integration_test

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/openapi"
)

func TestCreateDataproduct(t *testing.T) {
	resp, err := client.CreateDataproduct(context.Background(), openapi.CreateDataproductJSONRequestBody{
		Name: "new dataproduct",
		Owner: openapi.Owner{
			Team: auth.MockUser.Teams[0],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %v, got %v", http.StatusCreated, resp.StatusCode)
	}
}

func TestCreateDataproductUnauthorized(t *testing.T) {
	resp, err := client.CreateDataproduct(context.Background(), openapi.CreateDataproductJSONRequestBody{
		Name: "new dataproduct",
		Owner: openapi.Owner{
			Team: "invalid-team",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}
