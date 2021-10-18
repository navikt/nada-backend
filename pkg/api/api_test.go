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

func TestCreating_dataproduct(t *testing.T) {
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

	if in.Owner.Group != dataproduct.Owner.Group {
		t.Errorf("Expected group %q, but got %q", in.Owner.Group, dataproduct.Owner.Group)
	}

	if dataproduct.Id == "" {
		t.Error("Returned dataproduct has no ID")
	}

	if dataproduct.Name != in.Name {
		t.Errorf("Got name %q, want %q", dataproduct.Name, in.Name)
	}

	if dataproduct.Datasource == nil {
		t.Errorf("Got empty datasource")
	}
}

func TestCreating_dataproduct_for_other_team_is_not_authorized(t *testing.T) {
	in := newDataproduct()
	in.Owner.Group = "other-group"

	resp, err := client.CreateDataproduct(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestGetting_dataproduct(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	resp, err := client.GetDataproduct(context.Background(), existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Not 200 response")
	}

	var dp openapi.Dataproduct
	if err := json.NewDecoder(resp.Body).Decode(&dp); err != nil {
		t.Fatal(err)
	}

	if dp.Id != existing.Id {
		t.Errorf("Got id %q, want %q", dp.Id, existing.Id)
	}

	if dp.Name != existing.Name {
		t.Errorf("Got name %q, want %q", dp.Name, existing.Name)
	}

	if dp.Owner.Group != existing.Owner.Group {
		t.Errorf("Got group %q, want %q", dp.Owner.Group, existing.Owner.Group)
	}
}

func TestGetting_dataproducts(t *testing.T) {
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

	if dp.Owner.Group != existing.Owner.Group {
		t.Errorf("Got group %q, want %q", dp.Owner.Group, existing.Owner.Group)
	}
}

func TestUpdating_dataproduct(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	dp := openapi.UpdateDataproductJSONRequestBody{
		Name: "new name",
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

func TestUpdating_dataproduct_for_other_team_is_not_authorized(t *testing.T) {
	existing, err := repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
		Name: "dataproduct",
		Owner: openapi.Owner{
			Group: "other-group",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	update := openapi.UpdateDataproductJSONRequestBody{
		Name: "update",
	}

	resp, err := client.UpdateDataproduct(context.Background(), existing.Id, update)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestDeleting_dataproduct(t *testing.T) {
	existing := createDataproduct(newDataproduct())

	resp, err := client.DeleteDataproduct(context.Background(), existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code %v, got %v", http.StatusNoContent, resp.StatusCode)
	}
}

func TestDeleting_other_teams_dataproduct_is_not_authorized(t *testing.T) {
	existing, err := repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
		Name: "dataproduct",
		Owner: openapi.Owner{
			Group: "other-group",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.DeleteDataproduct(context.Background(), existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func newDataproductCollection() openapi.CreateDataproductCollectionJSONRequestBody {
	return openapi.CreateDataproductCollectionJSONRequestBody{
		Name: "new dataproductcollection",
		Owner: openapi.Owner{
			Group: auth.MockUser.Groups[0].Email,
		},
	}
}

func newDataproduct() openapi.CreateDataproductJSONRequestBody {
	return openapi.CreateDataproductJSONRequestBody{
		Name: "My dataset",
		Pii:  true,
		Owner: openapi.Owner{
			Group: auth.MockUser.Groups[0].Email,
		},
		Datasource: openapi.Bigquery{
			ProjectId: auth.MockProjectIDs[0],
			Dataset:   "dataset",
			Table:     "table",
		},
	}
}

func createDataproductCollection(in openapi.CreateDataproductCollectionJSONRequestBody) openapi.DataproductCollection {
	resp, err := client.CreateDataproductCollection(context.Background(), in)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var ret openapi.DataproductCollection
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.Fatal(err)
	}

	return ret
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
