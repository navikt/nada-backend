//go:build integration_test

package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"testing"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/openapi"
)

func TestCreateDataproduct(t *testing.T) {
	in := newDataproduct()

	resp, err := client.CreateDataproduct(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %v, got %v", http.StatusCreated, resp.StatusCode)
	}

	var dataproduct openapi.Dataproduct
	if err := json.NewDecoder(resp.Body).Decode(&dataproduct); err != nil {
		t.Fatal(err)
	}

	if in.Name != dataproduct.Name {
		t.Errorf("Expected name %q, but got %q", in.Name, dataproduct.Name)
	}

	if in.Owner.Team != dataproduct.Owner.Team {
		t.Errorf("Expected team %q, but got %q", in.Owner.Team, dataproduct.Owner.Team)
	}

	if dataproduct.Id == "" {
		t.Error("Returned dataproduct has no ID")
	}
}

func TestCreateDataproduct_Unauthorized(t *testing.T) {
	in := newDataproduct()
	in.Owner.Team = "invalid-team"
	resp, err := client.CreateDataproduct(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestGetDataproduct(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	resp, err := client.GetDataproduct(context.Background(), existing.Id)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	var dp openapi.Dataproduct
	json.NewDecoder(resp.Body).Decode(&dp)

	if dp.Id != existing.Id {
		t.Errorf("Got id %q, want %q", dp.Id, existing.Id)
	}

	if dp.Name != existing.Name {
		t.Errorf("Got name %q, want %q", dp.Name, existing.Name)
	}

	if dp.Owner.Team != existing.Owner.Team {
		t.Errorf("Got team %q, want %q", dp.Owner.Team, existing.Owner.Team)
	}
}

func TestGetDataproducts(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	resp, err := client.GetDataproducts(context.Background(), &openapi.GetDataproductsParams{
		Limit:  intPtr(100),
		Offset: intPtr(0),
	})
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	var dps []openapi.Dataproduct
	json.NewDecoder(resp.Body).Decode(&dps)

	var dp openapi.Dataproduct
	for _, entry := range dps {
		if entry.Id == existing.Id {
			dp = entry
			break
		}
	}

	if dp.Id != existing.Id {
		t.Errorf("Got id %q, want %q", dp.Id, existing.Id)
	}

	if dp.Name != existing.Name {
		t.Errorf("Got name %q, want %q", dp.Name, existing.Name)
	}

	if dp.Owner.Team != existing.Owner.Team {
		t.Errorf("Got team %q, want %q", dp.Owner.Team, existing.Owner.Team)
	}
}

func TestUpdateDataproduct(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	dp := openapi.UpdateDataproductJSONRequestBody{
		Name: "new name",
		Owner: openapi.Owner{
			Team: "team",
		},
	}

	resp, err := client.UpdateDataproduct(context.Background(), existing.Id, dp)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out openapi.Dataproduct
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != dp.Name {
		t.Errorf("Got name %q, want %q", out.Name, dp.Name)
	}
}

func newDataproduct() openapi.CreateDataproductJSONRequestBody {
	return openapi.CreateDataproductJSONRequestBody{
		Name: "new dataproduct",
		Owner: openapi.Owner{
			Team: auth.MockUser.Teams[0],
		},
	}
}

func createDataproduct(in openapi.CreateDataproductJSONRequestBody) openapi.Dataproduct {
	resp, err := client.CreateDataproduct(context.Background(), in)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var ret openapi.Dataproduct
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.Fatal(err)
	}

	return ret
}

func intPtr(i int) *int {
	return &i
}
