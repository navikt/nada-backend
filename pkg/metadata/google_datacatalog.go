package metadata

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	datacatalog "cloud.google.com/go/datacatalog/apiv1"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"google.golang.org/api/iterator"
	datacatalogpb "google.golang.org/genproto/googleapis/cloud/datacatalog/v1"
)

type Datacatalog struct {
	client datacatalogwrapper
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
	return &Datacatalog{client: &catalogWrapper{client: client}}, nil
}

func (c *Datacatalog) Close() error {
	return c.client.Close()
}

func (c *Datacatalog) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (*bigquery.TableMetadata, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return client.Dataset(datasetID).Table(tableID).Metadata(ctx)
}

func (c *Datacatalog) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	datasets := []string{}
	it := client.Datasets(ctx)
	for {
		ds, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		datasets = append(datasets, ds.DatasetID)
	}
	return datasets, nil
}

func (c *Datacatalog) GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tables := []*models.BigQueryTable{}
	it := client.Dataset(datasetID).Tables(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		m, err := t.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		tables = append(tables, &models.BigQueryTable{
			Name:         t.TableID,
			Description:  m.Description,
			Type:         models.BigQueryType(strings.ToLower(string(m.Type))),
			LastModified: m.LastModifiedTime,
		})
	}
	return tables, nil
}

func (c *Datacatalog) GetDatasetSchema(ctx context.Context, ds gensql.DatasourceBigquery) (Schema, error) {
	resourceURI := fmt.Sprintf("//bigquery.googleapis.com/projects/%v/datasets/%v/tables/%v", ds.ProjectID, ds.Dataset, ds.TableName)
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

type datacatalogwrapper interface {
	Close() error
	SearchCatalog(ctx context.Context, req *datacatalogpb.SearchCatalogRequest) ([]*datacatalogpb.SearchCatalogResult, error)
	LookupEntry(ctx context.Context, req *datacatalogpb.LookupEntryRequest) (*datacatalogpb.Entry, error)
}

type catalogWrapper struct {
	client *datacatalog.Client
}

func (c *catalogWrapper) SearchCatalog(ctx context.Context, req *datacatalogpb.SearchCatalogRequest) ([]*datacatalogpb.SearchCatalogResult, error) {
	it := c.client.SearchCatalog(ctx, req)

	ret := []*datacatalogpb.SearchCatalogResult{}
	for {
		el, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ret = append(ret, el)
	}

	return ret, nil
}

func (c *catalogWrapper) LookupEntry(ctx context.Context, req *datacatalogpb.LookupEntryRequest) (*datacatalogpb.Entry, error) {
	return c.client.LookupEntry(ctx, req)
}

func (c *catalogWrapper) Close() error {
	return c.client.Close()
}
