//go:build integration_test

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/openapi"
)

func TestCreateDataproduct(t *testing.T) {
	client := server.Client()

	body := &bytes.Buffer{}
	json.NewEncoder(body).Encode(openapi.NewDataproduct{
		Name: "new dataproduct",
		Owner: openapi.Owner{
			Team: auth.MockUser.Teams[0],
		},
	})

	resp, err := client.Post(server.URL+"/api/dataproducts", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %v, got %v", http.StatusCreated, resp.StatusCode)
	}
}

func TestCreateDataproductUnauthorized(t *testing.T) {
	client := server.Client()

	body := &bytes.Buffer{}
	json.NewEncoder(body).Encode(openapi.NewDataproduct{
		Name: "new dataproduct",
		Owner: openapi.Owner{
			Team: "invalid-team",
		},
	})

	resp, err := client.Post(server.URL+"/api/dataproducts", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}
