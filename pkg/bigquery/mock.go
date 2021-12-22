package bigquery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Mock struct {
	Tables []*models.BigQueryTable
}

func NewMock() *Mock {
	return &Mock{
		Tables: []*models.BigQueryTable{
			{
				Name:         "table1",
				Description:  "description1",
				Type:         models.BigQueryTypeTable,
				LastModified: time.Now().Add(-300 * time.Hour),
			},
			{
				Name:         "table2",
				Description:  "description2",
				Type:         models.BigQueryTypeTable,
				LastModified: time.Now(),
			},
			{
				Name:         "view1",
				Description:  "description1",
				Type:         models.BigQueryTypeView,
				LastModified: time.Now().Add(-20 * time.Hour),
			},
		},
	}
}

func (m *Mock) GetTables(ctx context.Context, projectID, datasetID string) ([]*models.BigQueryTable, error) {
	return m.Tables, nil
}

func (m *Mock) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
	return []string{"dataset1", "dataset2"}, nil
}

func (m *Mock) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (models.BigqueryMetadata, error) {
	var table *models.BigQueryTable
	for _, t := range m.Tables {
		if t.Name == tableID {
			table = t
			break
		}
	}

	if table == nil {
		return models.BigqueryMetadata{}, fmt.Errorf("mock table not found")
	}

	return models.BigqueryMetadata{
		TableType:    bigquery.TableType(strings.ToUpper(string(table.Type))),
		LastModified: table.LastModified,
	}, nil
}
