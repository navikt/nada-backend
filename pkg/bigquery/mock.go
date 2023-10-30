package bigquery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
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
		TableType:   bigquery.TableType(strings.ToUpper(string(table.Type))),
		Created:     time.Now(),
		Expires:     time.Date(2033, time.April, 14, 18, 00, 00, 00, time.UTC),
		Description: "This is a table description explaining the contents of the table",
		Schema: models.BigquerySchema{
			Columns: []models.BigqueryColumn{
				{
					Name: "test_column_1",
					Type: "STRING",
					Mode: "NULLABLE",
				},
				{
					Name:        "test_column_2",
					Type:        "INTEGER",
					Mode:        "",
					Description: "Some column description that explains this column",
				},
				{
					Name: "test_column_3",
					Type: "STRING",
					Mode: "NULLABLE",
				},
				{
					Name:        "test_column_4",
					Type:        "TIMESTAMP",
					Mode:        "",
					Description: "Some column description",
				},
				{
					Name: "test_column_5",
					Type: "FLOAT",
					Mode: "NULLABLE",
				},
				{
					Name:        "test_column_6",
					Type:        "INTEGER",
					Mode:        "",
					Description: "Some column description with more text than might be possible to apply, but might still be a good test for the frontend to receive",
				},
			},
		},
		LastModified: table.LastModified,
	}, nil
}

func (m *Mock) CreatePseudonymisedView(ctx context.Context, projectID, datasetID, tableID string, piiColumns []string) (string, string, string, error) {
	return "p", "d", "t", nil
}

func (c *Mock) CreateJoinableViewsForUser(ctx context.Context, user string, tableUrls []models.BigQuery) (string, string, map[uuid.UUID]string, error) {
	return "", "", nil, nil
}

func (c *Mock) GetJoinableViewsForUser(ctx context.Context, user *auth.User) ([]*models.JoinableView, error) {
	return nil, nil
}
