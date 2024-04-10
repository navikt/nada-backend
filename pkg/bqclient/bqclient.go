package bqclient

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type BigQueryType string

type BigquerySchema struct {
	Columns []BigqueryColumn `json:"columns"`
}

type BigqueryMetadata struct {
	Schema       BigquerySchema     `json:"schema"`
	TableType    bigquery.TableType `json:"tableType"`
	LastModified time.Time          `json:"lastModified"`
	Created      time.Time          `json:"created"`
	Expires      time.Time          `json:"expires"`
	Description  string             `json:"description"`
}

type BigqueryColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	Description string `json:"description"`
}

const (
	BigQueryTypeTable            BigQueryType = "table"
	BigQueryTypeView             BigQueryType = "view"
	BigQueryTypeMaterializedView BigQueryType = "materialized_view"
)

var AllBigQueryType = []BigQueryType{
	BigQueryTypeTable,
	BigQueryTypeView,
	BigQueryTypeMaterializedView,
}

func (e BigQueryType) IsValid() bool {
	switch e {
	case BigQueryTypeTable, BigQueryTypeView, BigQueryTypeMaterializedView:
		return true
	}
	return false
}

func (e BigQueryType) String() string {
	return string(e)
}

func (e *BigQueryType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BigQueryType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BigQueryType", str)
	}
	return nil
}

func (e BigQueryType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type BigQueryTable struct {
	Description  string       `json:"description"`
	LastModified time.Time    `json:"lastModified"`
	Name         string       `json:"name"`
	Type         BigQueryType `json:"type"`
}

func isSupportedTableType(tableType bigquery.TableType) bool {
	// We only support regular tables, views and materialized views for now.
	supported := []bigquery.TableType{
		bigquery.RegularTable,
		bigquery.ViewTable,
		bigquery.MaterializedView,
	}

	for _, tt := range supported {
		if tt == tableType {
			return true
		}
	}

	return false
}

func GetTables(ctx context.Context, projectID, datasetID string) ([]*BigQueryTable, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tables := []*BigQueryTable{}
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

		if !isSupportedTableType(m.Type) {
			continue
		}

		tables = append(tables, &BigQueryTable{
			Name:         t.TableID,
			Description:  m.Description,
			Type:         BigQueryType(strings.ToLower(string(m.Type))),
			LastModified: m.LastModifiedTime,
		})
	}

	return tables, nil
}

func GetDatasets(ctx context.Context, projectID string) ([]string, error) {
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

func TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (BigqueryMetadata, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return BigqueryMetadata{}, err
	}

	m, err := client.Dataset(datasetID).Table(tableID).Metadata(ctx)
	if err != nil {
		return BigqueryMetadata{}, err
	}

	schema := BigquerySchema{}

	for _, c := range m.Schema {
		ct := "NULLABLE"
		switch {
		case c.Repeated:
			ct = "REPEATED"
		case c.Required:
			ct = "REQUIRED"
		}
		schema.Columns = append(schema.Columns, BigqueryColumn{
			Name:        c.Name,
			Type:        string(c.Type),
			Mode:        ct,
			Description: c.Description,
		})
	}

	metadata := BigqueryMetadata{
		Schema:       schema,
		LastModified: m.LastModifiedTime,
		Created:      m.CreationTime,
		Expires:      m.ExpirationTime,
		TableType:    m.Type,
		Description:  m.Description,
	}

	return metadata, nil
}
