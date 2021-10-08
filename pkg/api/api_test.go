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

	if in.Owner.Team != dataproduct.Owner.Team {
		t.Errorf("Expected team %q, but got %q", in.Owner.Team, dataproduct.Owner.Team)
	}

	if dataproduct.Id == "" {
		t.Error("Returned dataproduct has no ID")
	}
}

func TestCreating_dataproduct_for_other_team_is_not_authorized(t *testing.T) {
	in := newDataproduct()
	in.Owner.Team = "other-team"

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

	if dp.Owner.Team != existing.Owner.Team {
		t.Errorf("Got team %q, want %q", dp.Owner.Team, existing.Owner.Team)
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

	if dp.Owner.Team != existing.Owner.Team {
		t.Errorf("Got team %q, want %q", dp.Owner.Team, existing.Owner.Team)
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
			Team: "other-team",
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
			Team: "other-team",
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

func TestCreating_dataset(t *testing.T) {
	existingDp := createDataproduct(newDataproduct())
	ds := newDataset(existingDp.Id)

	resp, err := client.CreateDataset(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code %v, got %v", http.StatusCreated, resp.StatusCode)
	}

	var created openapi.Dataset
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	if created.Name != ds.Name {
		t.Errorf("Got name %q, want %q", created.Name, ds.Name)
	}

	if created.Bigquery != ds.Bigquery {
		t.Errorf("Got bigquery %q, want %q", created.Name, ds.Name)
	}
}

func TestCreating_other_teams_dataset_is_not_authorized(t *testing.T) {
	existingDp := createDataproduct(newDataproduct())

	ds := openapi.CreateDatasetJSONRequestBody{
		Name:          "My dataset",
		DataproductId: existingDp.Id,
		Bigquery: openapi.BigQuery{
			ProjectId: "other-team-project-id",
			Dataset:   "dataset",
			Table:     "table",
		},
	}

	resp, err := client.CreateDataset(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestUpdating_dataset(t *testing.T) {
	existingDp := createDataproduct(newDataproduct())
	ds := newDataset(existingDp.Id)
	existingDs := createDataset(ds)

	updatedReq := openapi.UpdateDatasetJSONRequestBody{
		Name:          "updated name",
		DataproductId: existingDs.DataproductId,
		Bigquery: openapi.BigQuery{
			ProjectId: ds.Bigquery.ProjectId,
			Dataset:   ds.Bigquery.Dataset,
			Table:     ds.Bigquery.Table,
		},
	}

	resp, err := client.UpdateDataset(context.Background(), existingDs.Id, updatedReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, resp.StatusCode)
	}

	var updated openapi.Dataset
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}

	if updated.Name != updatedReq.Name {
		t.Errorf("Got name %q, want %q", updated.Name, ds.Name)
	}
}

func TestUpdating_other_teams_dataset_is_not_authorized(t *testing.T) {
	existingDp := createDataproduct(newDataproduct())
	ds := newDataset(existingDp.Id)
	existingDs := createDataset(ds)

	updatedReq := openapi.UpdateDatasetJSONRequestBody{
		Name:          ds.Name,
		DataproductId: existingDs.DataproductId,
		Bigquery: openapi.BigQuery{
			ProjectId: "other-team-project-id",
			Dataset:   ds.Bigquery.Dataset,
			Table:     ds.Bigquery.Table,
		},
	}

	resp, err := client.UpdateDataset(context.Background(), existingDs.Id, updatedReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestDeleting_dataset(t *testing.T) {
	existingDp := createDataproduct(newDataproduct())
	ds := newDataset(existingDp.Id)
	existingDs := createDataset(ds)

	resp, err := client.DeleteDataset(context.Background(), existingDs.Id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status code %v, got %v", http.StatusNoContent, resp.StatusCode)
	}
}

func TestDeleting_other_teams_dataset_is_not_authorized(t *testing.T) {
	existingDp, err := repo.CreateDataproduct(context.Background(), openapi.NewDataproduct{
		Name: "dataproduct",
		Owner: openapi.Owner{
			Team: "other-team",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ds, err := repo.CreateDataset(context.Background(), openapi.NewDataset{
		Name:          "dataset",
		DataproductId: existingDp.Id,
		Bigquery: openapi.BigQuery{
			ProjectId: "other-teams-project-id",
			Dataset:   "dataset",
			Table:     "table",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.DeleteDataset(context.Background(), ds.Id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %v, got %v", http.StatusNoContent, resp.StatusCode)
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

func newDataset(dpID string) openapi.CreateDatasetJSONRequestBody {
	return openapi.CreateDatasetJSONRequestBody{
		Name:          "My dataset",
		DataproductId: dpID,
		Bigquery: openapi.BigQuery{
			ProjectId: auth.MockProjectIDs[0],
			Dataset:   "dataset",
			Table:     "table",
		},
	}
}

func createDataset(in openapi.CreateDatasetJSONRequestBody) openapi.Dataset {
	resp, err := client.CreateDataset(context.Background(), in)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var ret openapi.Dataset
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
