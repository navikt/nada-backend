package metadata

import (
	"context"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"testing"

	"github.com/google/go-cmp/cmp"
	datacatalogpb "google.golang.org/genproto/googleapis/cloud/datacatalog/v1"
)

func TestGoogleDatacatalog(t *testing.T) {
	expected := []gensql.DatasourceBigquery{
		{
			ProjectID: "project_id",
			Dataset:   "mydataset",
			TableName: "mytable",
		},
	}
	dcc := &googleMockClient{
		searchResponse: []*datacatalogpb.SearchCatalogResult{
			{LinkedResource: "//some/ completely /// other / type / of / resource"},
			{LinkedResource: "//bigquery.googleapis.com/projects/project_id/datasets/mydataset/tables/mytable"},
			{LinkedResource: "//bigquery.googleapis.com/projects/project_id/datasets/mydataset"},
		},
	}
	client := &Datacatalog{client: dcc}

	res, err := client.GetDatasets(context.Background(), "project_id")
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(res, expected) {
		t.Error(cmp.Diff(res, expected))
	}
}

type googleMockClient struct {
	searchResponse []*datacatalogpb.SearchCatalogResult
	lookupResponse *datacatalogpb.Entry
	err            error
}

func (g *googleMockClient) SearchCatalog(ctx context.Context, req *datacatalogpb.SearchCatalogRequest) ([]*datacatalogpb.SearchCatalogResult, error) {
	return g.searchResponse, g.err
}

func (g *googleMockClient) LookupEntry(ctx context.Context, req *datacatalogpb.LookupEntryRequest) (*datacatalogpb.Entry, error) {
	return g.lookupResponse, g.err
}

func (g *googleMockClient) Close() error { return nil }
