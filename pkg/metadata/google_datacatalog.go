package metadata

import (
	"context"
	"fmt"
	"strings"
	"time"

	datacatalog "cloud.google.com/go/datacatalog/apiv1"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
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

func (c *Datacatalog) GetDatasets(ctx context.Context, projectID string) ([]gensql.DatasourceBigquery, error) {
	results, err := c.client.SearchCatalog(ctx, &datacatalogpb.SearchCatalogRequest{
		Scope: &datacatalogpb.SearchCatalogRequest_Scope{
			IncludeProjectIds: []string{projectID},
		},
		Query: "system=BIGQUERY projectid=" + projectID,
	})
	if err != nil {
		return nil, err
	}

	ret := []gensql.DatasourceBigquery{}

	for _, result := range results {
		parts := strings.Split(result.GetLinkedResource(), "/")
		if len(parts) != 9 {
			continue
		}
		dataset := parts[6]
		table := parts[8]
		ret = append(ret, gensql.DatasourceBigquery{
			ProjectID: projectID,
			Dataset:   dataset,
			TableName: table,
		})
	}

	return ret, nil
}

func (c *Datacatalog) GetDataset(ctx context.Context, projectID, datasetID string) ([]openapi.BigqueryTypeMetadata, error) {
	logrus.Info("we're in!")
	results, err := c.client.SearchCatalog(ctx, &datacatalogpb.SearchCatalogRequest{
		Scope: &datacatalogpb.SearchCatalogRequest_Scope{
			IncludeProjectIds: []string{projectID},
		},
		Query: fmt.Sprintf("system=BIGQUERY parent=%v.%v type=table or type=view", projectID, datasetID),
	})
	if err != nil {
		return nil, err
	}

	ret := []openapi.BigqueryTypeMetadata{}

	for _, result := range results {
		parts := strings.Split(result.GetLinkedResource(), "/")
		logrus.Info(parts)
		if len(parts) != 9 {
			continue
		}
		name := ""
		bqType := parts[6]
		description := parts[8]
		lastModified := time.Now()
		ret = append(ret, openapi.BigqueryTypeMetadata{
			Name:         name,
			Type:         openapi.BigqueryType(bqType),
			Description:  description,
			LastModified: lastModified,
		})
	}

	return ret, nil
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
