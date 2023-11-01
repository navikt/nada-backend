package bigquery

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/navikt/nada-backend/pkg/utils"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

type Bigquery struct {
	centralDataProject string
	pseudoDataset      string
}

func New(ctx context.Context, centralDataProject, pseudoDataset string) (*Bigquery, error) {
	return &Bigquery{
		centralDataProject: centralDataProject,
		pseudoDataset:      pseudoDataset,
	}, nil
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
		Description:  m.Description,
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

		if !isSupportedTableType(m.Type) {
			continue
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

func (c *Bigquery) ComposePseudoViewQuery(projectID, datasetID, tableID string, targetColumns []string) string {
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

func (c *Bigquery) CreatePseudonymisedView(ctx context.Context, projectID, datasetID, tableID string, piiColumns []string) (string, string, string, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return "", "", "", fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	if err := c.createDataset(ctx, projectID, c.pseudoDataset); err != nil {
		return "", "", "", fmt.Errorf("create pseudo dataset: %v", err)
	}

	viewQuery := c.ComposePseudoViewQuery(projectID, datasetID, tableID, piiColumns)
	meta := &bigquery.TableMetadata{
		ViewQuery: viewQuery,
	}
	pseudoViewID := fmt.Sprintf("%v_%v", datasetID, tableID)
	if err := client.Dataset(c.pseudoDataset).Table(pseudoViewID).Create(ctx, meta); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
			prevMeta, err := client.Dataset(c.pseudoDataset).Table(pseudoViewID).Metadata(ctx)
			if err != nil {
				return "", "", "", fmt.Errorf("Failed to fetch existing view metadata: %v", err)
			}
			_, err = client.Dataset(c.pseudoDataset).Table(pseudoViewID).Update(ctx, bigquery.TableMetadataToUpdate{ViewQuery: viewQuery}, prevMeta.ETag)
			if err != nil {
				return "", "", "", fmt.Errorf("Failed to update existing view: %v", err)
			}
		} else {
			return "", "", "", err
		}
	}

	return projectID, c.pseudoDataset, pseudoViewID, nil
}

func (c *Bigquery) ComposeJoinableViewQuery(plainTableUrl models.BigQuery, joinableDatasetID string, pseudoColumns []string) string {
	qSalt := fmt.Sprintf("WITH unified_salt AS (SELECT value AS salt FROM `%v.%v.%v` ds WHERE ds.key='%v')", c.centralDataProject, "secrets_vault", "secrets", joinableDatasetID)

	qSelect := "SELECT "
	for _, c := range pseudoColumns {
		qSelect += fmt.Sprintf(" SHA256(%v || unified_salt.salt) AS _x_%v", c, c)
		qSelect += ","
	}

	qSelect += "I.* EXCEPT("

	for i, c := range pseudoColumns {
		qSelect += c
		if i != len(pseudoColumns)-1 {
			qSelect += ","
		} else {
			qSelect += ")"
		}
	}
	qFrom := fmt.Sprintf("FROM `%v.%v.%v` AS I, unified_salt", plainTableUrl.ProjectID, plainTableUrl.Dataset, plainTableUrl.Table)

	return qSalt + " " + qSelect + " " + qFrom
}

func (c *Bigquery) CreateJoinableView(ctx context.Context, joinableDatasetID string, refDatasource models.BigQuery) (string, error) {
	query := c.ComposeJoinableViewQuery(refDatasource, joinableDatasetID, refDatasource.PseudoColumns)

	centralProjectclient, err := bigquery.NewClient(ctx, c.centralDataProject)
	if err != nil {
		return "", fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer centralProjectclient.Close()

	joinableViewMeta := &bigquery.TableMetadata{
		ViewQuery: query,
	}

	tableID := utils.MakeJoinableViewName(refDatasource.ProjectID, refDatasource.Dataset, refDatasource.Table)

	if err := centralProjectclient.Dataset(joinableDatasetID).Table(tableID).Create(ctx, joinableViewMeta); err != nil {
		return "", fmt.Errorf("Failed to create joinable view, please make sure the datasets are located in europe-north1 region in google cloud: %v", err)
	}

	return tableID, nil
}

func (c *Bigquery) CreateJoinableViewsForUser(ctx context.Context, name string, tableUrls []models.BigQuery) (string, string, map[uuid.UUID]string, error) {
	client, err := bigquery.NewClient(ctx, c.centralDataProject)
	if err != nil {
		return "", "", nil, fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	joinableDatasetID, err := c.createDatasetInCentralProject(ctx, name)
	if err != nil {
		return "", "", nil, err
	}
	c.createSecretTable(ctx, "secrets_vault", "secrets")
	c.insertSecretIfNotExists(ctx, "secrets_vault", "secrets", joinableDatasetID)

	viewsMap := map[uuid.UUID]string{}
	for _, table := range tableUrls {
		if v, err := c.CreateJoinableView(ctx, joinableDatasetID, table); err != nil {
			return "", "", nil, err
		} else {
			viewsMap[table.DatasetID] = v
		}
	}

	return c.centralDataProject, joinableDatasetID, viewsMap, nil
}

func (c *Bigquery) createDataset(ctx context.Context, projectID, datasetID string) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	meta := &bigquery.DatasetMetadata{
		Location: "europe-north1", //TODO: we can support other regions
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

func (c *Bigquery) createDatasetInCentralProject(ctx context.Context, datasetID string) (string, error) {
	client, err := bigquery.NewClient(ctx, c.centralDataProject)
	if err != nil {
		return "", fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	meta := &bigquery.DatasetMetadata{
		Location: "europe-north1",
	}

	postfix := ""
	for i := 0; ; i++ {
		if i > 0 {
			postfix = fmt.Sprintf("%v", i)
		}
		err := client.Dataset(datasetID+postfix).Create(ctx, meta)
		if err == nil {
			break
		}

		if gerr, ok := err.(*googleapi.Error); !ok || gerr.Code != 409 {
			return "", err
		}
	}
	return datasetID + postfix, nil
}

func (c *Bigquery) createSecretTable(ctx context.Context, datasetID, tableID string) error {
	client, err := bigquery.NewClient(ctx, c.centralDataProject)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	meta := &bigquery.DatasetMetadata{
		Location: "europe-north1",
	}

	if err := client.Dataset("secrets_vault").Create(ctx, meta); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code != 409 {
			return fmt.Errorf("failed to create secret dataset: %v", err)
		}
	}

	sampleSchema := bigquery.Schema{
		{Name: "key", Type: bigquery.StringFieldType},
		{Name: "value", Type: bigquery.StringFieldType},
	}

	metaData := &bigquery.TableMetadata{
		Schema: sampleSchema,
	}
	tableRef := client.Dataset(datasetID).Table(tableID)
	if err := tableRef.Create(ctx, metaData); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
			return nil
		}
		return err
	}
	return nil
}

func (c *Bigquery) insertSecretIfNotExists(ctx context.Context, secretDatasetID, secretTableID, key string) error {
	client, err := bigquery.NewClient(ctx, c.centralDataProject)
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	encryptionKey, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	var insertQuery strings.Builder
	fmt.Fprintf(&insertQuery, "INSERT INTO `%v.%v.%v` (key, value) ", c.centralDataProject, secretDatasetID, secretTableID)
	fmt.Fprintf(&insertQuery, "SELECT '%v', '%v' FROM UNNEST([1]) ", key, encryptionKey.String())
	fmt.Fprintf(&insertQuery, "WHERE NOT EXISTS (SELECT 1 FROM `%v.%v.%v` WHERE key = '%v')", c.centralDataProject, secretDatasetID, secretTableID, key)

	job, err := client.Query(insertQuery.String()).Run(ctx)
	if err != nil {
		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}
	if status.Err() != nil {
		return err
	}

	return nil
}

func makeUserPrefix(user *auth.User) string {
	emailFixDot := strings.ReplaceAll(user.Email, ".", "_")
	emailFixAt := strings.ReplaceAll(emailFixDot, "@", "_")
	return emailFixAt
}
