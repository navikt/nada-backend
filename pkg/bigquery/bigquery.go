package bigquery

import (
	"context"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"google.golang.org/api/iterator"
)

type Bigquery struct{}

func New(ctx context.Context) (*Bigquery, error) {
	return &Bigquery{}, nil
}

func (c *Bigquery) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (models.BigqueryMetadata, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return models.BigqueryMetadata{}, err
	}

	m, err := client.Dataset(datasetID).Table(tableID).Metadata(ctx)
	if err != nil {
		return models.BigqueryMetadata{}, err
	}

	schema := models.BigquerySchema{}

	for _, c := range m.Schema {
		ct := "NULLABLE"
		switch {
		case c.Repeated:
			ct = "REPEATED"
		case c.Required:
			ct = "REQUIRED"
		}
		schema.Columns = append(schema.Columns, models.BigqueryColumn{
			Name:        c.Name,
			Type:        string(c.Type),
			Mode:        ct,
			Description: c.Description,
		})
	}

	metadata := models.BigqueryMetadata{
		Schema:       schema,
		LastModified: m.LastModifiedTime,
		Created:      m.CreationTime,
		Expires:      m.ExpirationTime,
		TableType:    m.Type,
	}

	return metadata, nil
}

func (c *Bigquery) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
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

func (c *Bigquery) GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error) {
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
