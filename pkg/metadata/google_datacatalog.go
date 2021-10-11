package metadata

import (
	"context"
	"fmt"

	datacatalog "cloud.google.com/go/datacatalog/apiv1"
	"github.com/navikt/nada-backend/pkg/openapi"
	datacatalogpb "google.golang.org/genproto/googleapis/cloud/datacatalog/v1"
)

//type CatalogClient interface {
//	GetSchema() (Schema, error)
//	GetUserBQAssets(user string) struct{}
//}

type Datacatalog struct {
	client *datacatalog.Client
}

type Schema struct {
	Columns []Column
}

type Column struct {
	Name        string
	Type        string
	Mode        string
	Description string
}

func NewDatacatalog(ctx context.Context) (*Datacatalog, error) {
	client, err := datacatalog.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiating datacatalog client: %w", err)
	}
	return &Datacatalog{client: client}, nil
}

func (c *Datacatalog) Close() error {
	return c.client.Close()
}

func (c *Datacatalog) GetDatasetSchema(ctx context.Context, ds openapi.BigQuery) (Schema, error) {
	resourceURI := fmt.Sprintf("//bigquery.googleapis.com/projects/%v/datasets/%v/tables/%v", ds.ProjectId, ds.Dataset, ds.Table)
	req := &datacatalogpb.LookupEntryRequest{
		TargetName: &datacatalogpb.LookupEntryRequest_LinkedResource{
			LinkedResource: resourceURI,
		},
	}

	resp, err := c.client.LookupEntry(ctx, req)
	if err != nil {
		return Schema{}, fmt.Errorf("looking up entry: %w", err)
	}

	schema := Schema{}

	for _, c := range resp.Schema.Columns {
		schema.Columns = append(schema.Columns, Column{
			Name:        c.Column,
			Type:        c.Type,
			Mode:        c.Mode,
			Description: c.Description,
		})
	}

	return schema, err
}
