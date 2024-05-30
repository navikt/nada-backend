package bqclient

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

type BigqueryClient struct {
	centralDataProject string
	pseudoDataset      string
}

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

func New(ctx context.Context, centralDataProject, pseudoDataset string) (*BigqueryClient, error) {
	return &BigqueryClient{
		centralDataProject: centralDataProject,
		pseudoDataset:      pseudoDataset,
	}, nil
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

func (b *BigqueryClient) GetTables(ctx context.Context, projectID, datasetID string) ([]*BigQueryTable, error) {
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

func (b *BigqueryClient) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
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

func (b *BigqueryClient) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (BigqueryMetadata, error) {
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

func composePseudoViewQuery(projectID, datasetID, tableID string, targetColumns []string) string {
	qGenSalt := `WITH gen_salt AS (
		SELECT GENERATE_UUID() AS salt
	)`

	qSelect := "SELECT "
	for _, c := range targetColumns {
		qSelect += fmt.Sprintf(" SHA256(%v || gen_salt.salt) AS %v", c, c)
		qSelect += ","
	}

	qSelect += "I.* EXCEPT("

	for i, c := range targetColumns {
		qSelect += c
		if i != len(targetColumns)-1 {
			qSelect += ","
		} else {
			qSelect += ")"
		}
	}
	qFrom := fmt.Sprintf("FROM `%v.%v.%v` AS I, gen_salt", projectID, datasetID, tableID)

	return qGenSalt + " " + qSelect + " " + qFrom
}

func createDataset(ctx context.Context, projectID, datasetID string) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	meta := &bigquery.DatasetMetadata{
		Location: "europe-north1", // TODO: we can support other regions
	}

	if err := client.Dataset(datasetID).Create(ctx, meta); err != nil {
		if err != nil {
			if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
				return nil
			}
			return err
		}
	}
	return nil
}

func (b *BigqueryClient) CreatePseudonymisedView(ctx context.Context, projectID, datasetID, tableID string, piiColumns []string) (string, string, string, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return "", "", "", fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	if err := createDataset(ctx, projectID, b.pseudoDataset); err != nil {
		return "", "", "", fmt.Errorf("create pseudo dataset: %v", err)
	}

	viewQuery := composePseudoViewQuery(projectID, datasetID, tableID, piiColumns)
	meta := &bigquery.TableMetadata{
		ViewQuery: viewQuery,
	}
	pseudoViewID := fmt.Sprintf("%v_%v", datasetID, tableID)
	if err := client.Dataset(b.pseudoDataset).Table(pseudoViewID).Create(ctx, meta); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
			prevMeta, err := client.Dataset(b.pseudoDataset).Table(pseudoViewID).Metadata(ctx)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to fetch existing view metadata: %v", err)
			}
			_, err = client.Dataset(b.pseudoDataset).Table(pseudoViewID).Update(ctx, bigquery.TableMetadataToUpdate{ViewQuery: viewQuery}, prevMeta.ETag)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to update existing view: %v", err)
			}
		} else {
			return "", "", "", err
		}
	}

	return projectID, b.pseudoDataset, pseudoViewID, nil
}

func (b *BigqueryClient) GetTableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (BigqueryMetadata, error) {
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
