package gcp

import (
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/iam"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"strings"
	"time"
)

type bigQueryAPI struct {
	endpoint      string
	gcpProject    string
	pseudoDataSet string
}

var _ service.BigQueryAPI = &bigQueryAPI{}

func (a *bigQueryAPI) GetTables(ctx context.Context, projectID, datasetID string) ([]*service.BigQueryTable, error) {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var tables []*service.BigQueryTable
	it := client.Dataset(datasetID).Tables(ctx)
	for {
		t, err := it.Next()
		if errors.Is(err, iterator.Done) {
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

		tables = append(tables, &service.BigQueryTable{
			Name:         t.TableID,
			Description:  m.Description,
			Type:         service.BigQueryType(strings.ToLower(string(m.Type))),
			LastModified: m.LastModifiedTime,
		})
	}

	return tables, nil
	// TODO implement me
	panic("implement me")
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

func (a *bigQueryAPI) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var datasets []string
	it := client.Datasets(ctx)
	for {
		ds, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		datasets = append(datasets, ds.DatasetID)
	}

	return datasets, nil
}

func (a *bigQueryAPI) createDataset(ctx context.Context, projectID, datasetID string) error {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()

	// FIXME: make this configurable
	meta := &bigquery.DatasetMetadata{
		Location: "europe-north1", // TODO: we can support other regions
	}

	if err := client.Dataset(datasetID).Create(ctx, meta); err != nil {
		if err != nil {
			var gerr *googleapi.Error
			if errors.As(err, &gerr) && gerr.Code == 409 {
				return nil
			}
			return err
		}
	}
	return nil
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

func (a *bigQueryAPI) CreatePseudonymisedView(ctx context.Context, projectID, datasetID, tableID string, piiColumns []string) (string, string, string, error) {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return "", "", "", err
	}
	defer client.Close()

	if err := a.createDataset(ctx, projectID, a.pseudoDataSet); err != nil {
		return "", "", "", fmt.Errorf("create pseudo dataset: %v", err)
	}

	viewQuery := composePseudoViewQuery(projectID, datasetID, tableID, piiColumns)
	meta := &bigquery.TableMetadata{
		ViewQuery: viewQuery,
	}
	pseudoViewID := fmt.Sprintf("%v_%v", datasetID, tableID)
	if err := client.Dataset(a.pseudoDataSet).Table(pseudoViewID).Create(ctx, meta); err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 409 {
			prevMeta, err := client.Dataset(a.pseudoDataSet).Table(pseudoViewID).Metadata(ctx)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to fetch existing view metadata: %v", err)
			}
			_, err = client.Dataset(a.pseudoDataSet).Table(pseudoViewID).Update(ctx, bigquery.TableMetadataToUpdate{ViewQuery: viewQuery}, prevMeta.ETag)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to update existing view: %v", err)
			}
		}
	}

	return projectID, a.pseudoDataSet, pseudoViewID, nil
}

func (a *bigQueryAPI) deleteBigqueryTable(ctx context.Context, projectID, datasetID, tableID string) error {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Dataset(datasetID).Table(tableID).Delete(ctx); err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 404 {
			return nil
		}

		return err
	}

	return nil
}

func (a *bigQueryAPI) DeleteJoinableView(ctx context.Context, joinableViewName, refProjectID, refDatasetID, refTableID string) error {
	return a.deleteBigqueryTable(ctx, a.gcpProject, joinableViewName, makeJoinableViewName(refProjectID, refDatasetID, refTableID))
}

func (a *bigQueryAPI) DeletePseudoView(ctx context.Context, pseudoProjectID, pseudoDatasetID, pseudoTableID string) error {
	if pseudoDatasetID != a.pseudoDataSet {
		return fmt.Errorf("cannot delete pseudo view from dataset %v, not a markedsplassen dataset", pseudoDatasetID)
	}

	return a.deleteBigqueryTable(ctx, pseudoProjectID, pseudoDatasetID, pseudoTableID)
}

func (a *bigQueryAPI) DeleteJoinableDataset(ctx context.Context, datasetID string) error {
	client, err := bigquery.NewClient(ctx, a.gcpProject)
	if err != nil {
		return err
	}

	if err := client.Dataset(datasetID).DeleteWithContents(ctx); err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 404 {
			return nil
		}

		return err
	}

	return nil
}

func (a *bigQueryAPI) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (service.BigqueryMetadata, error) {
	client, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return service.BigqueryMetadata{}, err
	}

	m, err := client.Dataset(datasetID).Table(tableID).Metadata(ctx)
	if err != nil {
		return service.BigqueryMetadata{}, err
	}

	schema := service.BigquerySchema{}

	for _, c := range m.Schema {
		ct := "NULLABLE"
		switch {
		case c.Repeated:
			ct = "REPEATED"
		case c.Required:
			ct = "REQUIRED"
		}
		schema.Columns = append(schema.Columns, service.BigqueryColumn{
			Name:        c.Name,
			Type:        string(c.Type),
			Mode:        ct,
			Description: c.Description,
		})
	}

	metadata := service.BigqueryMetadata{
		Schema:       schema,
		LastModified: m.LastModifiedTime,
		Created:      m.CreationTime,
		Expires:      m.ExpirationTime,
		TableType:    m.Type,
		Description:  m.Description,
	}

	return metadata, nil
}

// FIXME: duplicated
func makeJoinableViewName(projectID, datasetID, tableID string) string {
	// datasetID will always be same markedsplassen dataset id
	return fmt.Sprintf("%v_%v", projectID, tableID)
}

func (a *bigQueryAPI) ComposeJoinableViewQuery(plainTable service.DatasourceForJoinableView, joinableDatasetID string) string {
	qSalt := fmt.Sprintf("WITH unified_salt AS (SELECT value AS salt FROM `%v.%v.%v` ds WHERE ds.key='%v')", a.gcpProject, "secrets_vault", "secrets", joinableDatasetID)

	qSelect := "SELECT "
	for _, c := range plainTable.PseudoColumns {
		qSelect += fmt.Sprintf(" SHA256(%v || unified_salt.salt) AS _x_%v", c, c)
		qSelect += ","
	}

	qSelect += "I.* EXCEPT("

	for i, c := range plainTable.PseudoColumns {
		qSelect += c
		if i != len(plainTable.PseudoColumns)-1 {
			qSelect += ","
		} else {
			qSelect += ")"
		}
	}
	qFrom := fmt.Sprintf("FROM `%v.%v.%v` AS I, unified_salt", plainTable.Project, plainTable.Dataset, plainTable.Table)

	return qSalt + " " + qSelect + " " + qFrom
}

func (a *bigQueryAPI) CreateJoinableView(ctx context.Context, joinableDatasetID string, datasource service.JoinableViewDatasource) (string, error) {
	query := a.ComposeJoinableViewQuery(*datasource.RefDatasource, joinableDatasetID)

	fmt.Println(query)
	centralProjectclient, err := a.clientFromProjectID(ctx, a.gcpProject)
	if err != nil {
		return "", err
	}
	defer centralProjectclient.Close()

	joinableViewMeta := &bigquery.TableMetadata{
		ViewQuery: query,
	}

	tableID := makeJoinableViewName(datasource.PseudoDatasource.Project, datasource.PseudoDatasource.Dataset, datasource.PseudoDatasource.Table)

	if err := centralProjectclient.Dataset(joinableDatasetID).Table(tableID).Create(ctx, joinableViewMeta); err != nil {
		return "", fmt.Errorf("failed to create joinable view, please make sure the datasets are located in europe-north1 region in google cloud: %v", err)
	}

	return tableID, nil
}

func (a *bigQueryAPI) CreateJoinableViewsForUser(ctx context.Context, name string, datasources []service.JoinableViewDatasource) (string, string, map[string]string, error) {
	client, err := a.clientFromProjectID(ctx, a.gcpProject)
	if err != nil {
		return "", "", nil, err
	}
	defer client.Close()

	joinableDatasetID, err := a.createDatasetInCentralProject(ctx, name)
	if err != nil {
		return "", "", nil, err
	}
	err = a.createSecretTable(ctx, "secrets_vault", "secrets")
	if err != nil {
		return "", "", nil, err
	}

	err = a.insertSecretIfNotExists(ctx, "secrets_vault", "secrets", joinableDatasetID)
	if err != nil {
		return "", "", nil, err
	}

	viewsMap := map[string]string{}
	for _, d := range datasources {
		if v, err := a.CreateJoinableView(ctx, joinableDatasetID, d); err != nil {
			return "", "", nil, err
		} else {
			viewsMap[d.RefDatasource.Dataset] = v
		}
	}

	return a.gcpProject, joinableDatasetID, viewsMap, nil
}

func (a *bigQueryAPI) createDatasetInCentralProject(ctx context.Context, datasetID string) (string, error) {
	client, err := a.clientFromProjectID(ctx, a.gcpProject)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// FIXME: make this configurable
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

func (a *bigQueryAPI) createSecretTable(ctx context.Context, datasetID, tableID string) error {
	client, err := a.clientFromProjectID(ctx, a.gcpProject)
	if err != nil {
		return err
	}
	defer client.Close()

	// FIXME: make this configurable
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

func (a *bigQueryAPI) insertSecretIfNotExists(ctx context.Context, secretDatasetID, secretTableID, key string) error {
	client, err := a.clientFromProjectID(ctx, a.gcpProject)
	if err != nil {
		return err
	}
	defer client.Close()

	encryptionKey, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	var insertQuery strings.Builder
	fmt.Fprintf(&insertQuery, "INSERT INTO `%v.%v.%v` (key, value) ", a.gcpProject, secretDatasetID, secretTableID)
	fmt.Fprintf(&insertQuery, "SELECT '%v', '%v' FROM UNNEST([1]) ", key, encryptionKey.String())
	fmt.Fprintf(&insertQuery, "WHERE NOT EXISTS (SELECT 1 FROM `%v.%v.%v` WHERE key = '%v')", a.gcpProject, secretDatasetID, secretTableID, key)

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

func (a *bigQueryAPI) MakeBigQueryUrlForJoinableViews(name, projectID, datasetID, tableID string) string {
	return fmt.Sprintf("%v.%v.%v", a.gcpProject, name, makeJoinableViewName(projectID, datasetID, tableID))
}

func (a *bigQueryAPI) Grant(ctx context.Context, projectID, datasetID, tableID, member string) error {
	bqClient, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	var entityType bigquery.EntityType
	switch strings.Split(member, ":")[0] {
	case "user", "serviceAccount":
		entityType = bigquery.UserEmailEntity
	case "group":
		entityType = bigquery.GroupEmailEntity
	}
	newEntry := &bigquery.AccessEntry{
		EntityType: entityType,
		Entity:     strings.Split(member, ":")[1],
		Role:       bigquery.AccessRole("roles/bigquery.metadataViewer"),
	}
	ds := bqClient.Dataset(datasetID)
	m, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("ds.Metadata: %w", err)
	}

	update := bigquery.DatasetMetadataToUpdate{
		Access: append(m.Access, newEntry),
	}
	if _, err := ds.Update(ctx, update, m.ETag); err != nil {
		return fmt.Errorf("ds.Update: %w", err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	policy.Add(member, iam.RoleName(role))

	bqTable := bqClient.Dataset(datasetID).Table(tableID)

	return bqTable.IAM().SetPolicy(ctx, policy)
}

func (a *bigQueryAPI) Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error {
	bqClient, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	policy.Remove(member, iam.RoleName(role))

	bqTable := bqClient.Dataset(datasetID).Table(tableID)
	return bqTable.IAM().SetPolicy(ctx, policy)
}

func (a *bigQueryAPI) AddToAuthorizedViews(ctx context.Context, srcProjectID, srcDataset, sinkProjectID, sinkDataset, sinkTable string) error {
	bqClient, err := a.clientFromProjectID(ctx, srcProjectID)
	if err != nil {
		return err
	}
	defer bqClient.Close()
	ds := bqClient.Dataset(srcDataset)
	m, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("ds.Metadata: %w", err)
	}

	if m.Access != nil {
		for _, e := range m.Access {
			if e != nil && e.EntityType == bigquery.ViewEntity && e.View != nil &&
				e.View.ProjectID == sinkProjectID && e.View.DatasetID == sinkDataset && e.View.TableID == sinkTable {
				return nil
			}
		}
	}

	newEntry := &bigquery.AccessEntry{
		EntityType: bigquery.ViewEntity,
		View: &bigquery.Table{
			ProjectID: sinkProjectID,
			DatasetID: sinkDataset,
			TableID:   sinkTable,
		},
	}

	update := bigquery.DatasetMetadataToUpdate{
		Access: append(m.Access, newEntry),
	}
	if _, err := ds.Update(ctx, update, m.ETag); err != nil {
		return err
	}

	return nil
}

func (a *bigQueryAPI) HasAccess(ctx context.Context, projectID, datasetID, tableID, member string) (bool, error) {
	bqClient, err := a.clientFromProjectID(ctx, projectID)
	if err != nil {
		return false, err
	}
	defer bqClient.Close()

	policy, err := getPolicy(ctx, bqClient, datasetID, tableID)
	if err != nil {
		return false, fmt.Errorf("getting policy for %v.%v in %v: %v", datasetID, tableID, projectID, err)
	}

	// no support for V3 for BigQuery yet, and no support for conditions
	role := "roles/bigquery.dataViewer"
	return policy.HasRole(member, iam.RoleName(role)), nil
}

func (a *bigQueryAPI) clientFromProjectID(ctx context.Context, projectID string) (*bigquery.Client, error) {
	var options []option.ClientOption

	if a.endpoint != "" {
		options = append(options, option.WithEndpoint(a.endpoint))
	}

	client, err := bigquery.NewClient(ctx, projectID, options...)
	if err != nil {
		return nil, fmt.Errorf("creating bigquery client: %w", err)
	}

	return client, nil
}

func getPolicy(ctx context.Context, bqclient *bigquery.Client, datasetID, tableID string) (*iam.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	dataset := bqclient.Dataset(datasetID)
	table := dataset.Table(tableID)
	policy, err := table.IAM().Policy(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting policy for %v.%v: %v", datasetID, tableID, err)
	}

	return policy, nil
}

func NewBigQueryAPI(gcpProject, endpoint, pseudoDataSet string) *bigQueryAPI {
	return &bigQueryAPI{
		endpoint:      endpoint,
		gcpProject:    gcpProject,
		pseudoDataSet: pseudoDataSet,
	}
}
