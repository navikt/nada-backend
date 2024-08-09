package gcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type bigQueryAPI struct {
	client        bq.Operations
	gcpProject    string
	gcpRegion     string
	pseudoDataSet string
}

const (
	secretsDatasetID = "secrets_vault"
	secretsTableID   = "secrets"
)

var _ service.BigQueryAPI = &bigQueryAPI{}

func (a *bigQueryAPI) GetTables(ctx context.Context, projectID, datasetID string) ([]*service.BigQueryTable, error) {
	const op errs.Op = "bigQueryAPI.GetTables"

	rawTables, err := a.client.GetTables(ctx, projectID, datasetID)
	if err != nil {
		return nil, errs.E(errs.IO, op, errs.Parameter(datasetID), err)
	}

	var tables []*service.BigQueryTable
	for _, t := range rawTables {
		if !isSupportedTableType(t.Type) {
			continue
		}

		tables = append(tables, &service.BigQueryTable{
			Name:         t.TableID,
			Description:  t.Description,
			Type:         service.BigQueryTableType(t.Type),
			LastModified: t.LastModified,
		})
	}

	return tables, nil
}

func isSupportedTableType(tableType bq.TableType) bool {
	// We only support regular tables, views and materialized views for now.
	supported := []bq.TableType{
		bq.RegularTable,
		bq.ViewTable,
		bq.MaterializedView,
	}

	for _, tt := range supported {
		if tt == tableType {
			return true
		}
	}

	return false
}

func (a *bigQueryAPI) GetDatasets(ctx context.Context, projectID string) ([]string, error) {
	const op errs.Op = "bigQueryAPI.GetDatasets"

	rawDatasets, err := a.client.GetDatasets(ctx, projectID)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	datasets := make([]string, len(rawDatasets))
	for i, ds := range rawDatasets {
		datasets[i] = ds.DatasetID
	}

	return datasets, nil
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
	const op errs.Op = "bigQueryAPI.CreatePseudonymisedView"

	err := a.client.CreateDatasetIfNotExists(ctx, projectID, a.pseudoDataSet, a.gcpRegion)
	if err != nil {
		return "", "", "", errs.E(errs.IO, op, err)
	}

	viewQuery := composePseudoViewQuery(projectID, datasetID, tableID, piiColumns)
	pseudoViewID := fmt.Sprintf("%v_%v", datasetID, tableID)

	_, err = a.client.CreateTableOrUpdate(ctx, &bq.Table{
		ProjectID: projectID,
		DatasetID: a.pseudoDataSet,
		TableID:   pseudoViewID,
		Location:  a.gcpRegion,
		ViewQuery: viewQuery,
	})
	if err != nil {
		return "", "", "", errs.E(errs.IO, op, err)
	}

	return projectID, a.pseudoDataSet, pseudoViewID, nil
}

func (a *bigQueryAPI) deleteBigqueryTable(ctx context.Context, projectID, datasetID, tableID string) error {
	const op errs.Op = "bigQueryAPI.deleteBigqueryTable"

	err := a.client.DeleteTable(ctx, projectID, datasetID, tableID)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *bigQueryAPI) DeleteJoinableView(ctx context.Context, joinableViewName, refProjectID, refDatasetID, refTableID string) error {
	const op errs.Op = "bigQueryAPI.DeleteJoinableView"

	err := a.deleteBigqueryTable(ctx, a.gcpProject, joinableViewName, makeJoinableViewName(refProjectID, refDatasetID, refTableID))
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (a *bigQueryAPI) DeletePseudoView(ctx context.Context, pseudoProjectID, pseudoDatasetID, pseudoTableID string) error {
	const op errs.Op = "bigQueryAPI.DeletePseudoView"

	if pseudoDatasetID != a.pseudoDataSet {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("cannot delete pseudo view from dataset %v, not a markedsplassen dataset", pseudoDatasetID))
	}

	err := a.deleteBigqueryTable(ctx, pseudoProjectID, pseudoDatasetID, pseudoTableID)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (a *bigQueryAPI) DeleteJoinableDataset(ctx context.Context, datasetID string) error {
	const op errs.Op = "bigQueryAPI.DeleteJoinableDataset"

	err := a.client.DeleteDataset(ctx, a.gcpProject, datasetID, true)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *bigQueryAPI) TableMetadata(ctx context.Context, projectID string, datasetID string, tableID string) (service.BigqueryMetadata, error) {
	const op errs.Op = "bigQueryAPI.TableMetadata"

	table, err := a.client.GetTable(ctx, projectID, datasetID, tableID)
	if err != nil {
		return service.BigqueryMetadata{}, errs.E(errs.IO, op, err)
	}

	schema := service.BigquerySchema{}

	for _, c := range table.Schema {
		schema.Columns = append(schema.Columns, &service.BigqueryColumn{
			Name:        c.Name,
			Type:        c.Type.String(),
			Mode:        c.Mode.String(),
			Description: c.Description,
		})
	}

	metadata := service.BigqueryMetadata{
		Schema:       schema,
		LastModified: table.LastModified,
		Created:      table.Created,
		Expires:      table.Expires,
		TableType:    service.BigQueryTableType(table.Type),
		Description:  table.Description,
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
	const op errs.Op = "bigQueryAPI.CreateJoinableView"

	query := a.ComposeJoinableViewQuery(*datasource.RefDatasource, joinableDatasetID)
	tableID := makeJoinableViewName(datasource.PseudoDatasource.Project, datasource.PseudoDatasource.Dataset, datasource.PseudoDatasource.Table)

	err := a.client.CreateTable(ctx, &bq.Table{
		ProjectID: a.gcpProject,
		DatasetID: joinableDatasetID,
		TableID:   tableID,
		Location:  a.gcpRegion,
		ViewQuery: query,
	})
	if err != nil {
		return "", errs.E(errs.IO, op, err)
	}

	return tableID, nil
}

func (a *bigQueryAPI) CreateJoinableViewsForUser(ctx context.Context, name string, datasources []service.JoinableViewDatasource) (string, string, map[string]string, error) {
	const op errs.Op = "bigQueryAPI.CreateJoinableViewsForUser"

	joinableDatasetID, err := a.createDatasetInCentralProject(ctx, name)
	if err != nil {
		return "", "", nil, errs.E(op, err)
	}

	err = a.createSecretTable(ctx)
	if err != nil {
		return "", "", nil, errs.E(op, err)
	}

	err = a.insertSecretIfNotExists(ctx, joinableDatasetID)
	if err != nil {
		return "", "", nil, errs.E(op, err)
	}

	viewsMap := map[string]string{}
	for _, d := range datasources {
		if v, err := a.CreateJoinableView(ctx, joinableDatasetID, d); err != nil {
			return "", "", nil, errs.E(op, err)
		} else {
			viewsMap[d.RefDatasource.Dataset] = v
		}
	}

	return a.gcpProject, joinableDatasetID, viewsMap, nil
}

func (a *bigQueryAPI) createDatasetInCentralProject(ctx context.Context, datasetID string) (string, error) {
	const op errs.Op = "bigQueryAPI.createDatasetInCentralProject"

	uniqueDatasetID := bq.DatasetNameWithRandomPostfix(datasetID)

	err := a.client.CreateDataset(ctx, a.gcpProject, uniqueDatasetID, a.gcpRegion)
	if err != nil {
		return "", errs.E(errs.IO, op, err)
	}

	return uniqueDatasetID, nil
}

func (a *bigQueryAPI) createSecretTable(ctx context.Context) error {
	const op errs.Op = "bigQueryAPI.createSecretTable"

	err := a.client.CreateDatasetIfNotExists(ctx, a.gcpProject, secretsDatasetID, a.gcpRegion)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	err = a.client.CreateTable(ctx, &bq.Table{
		ProjectID: a.gcpProject,
		DatasetID: secretsDatasetID,
		TableID:   secretsTableID,
		Location:  a.gcpRegion,
		Schema: []*bq.Column{
			{Name: "key", Type: bq.StringFieldType},
			{Name: "value", Type: bq.StringFieldType},
		},
	})
	if err != nil {
		if errors.Is(err, bq.ErrExist) {
			return nil
		}

		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *bigQueryAPI) insertSecretIfNotExists(ctx context.Context, key string) error {
	const op errs.Op = "bigQueryAPI.insertSecretIfNotExists"

	encryptionKey, err := uuid.NewUUID()
	if err != nil {
		return errs.E(errs.Internal, op, fmt.Errorf("generating encryption key: %w", err))
	}

	var insertQuery strings.Builder
	_, _ = fmt.Fprintf(&insertQuery, "INSERT INTO `%v.%v.%v` (key, value) ", a.gcpProject, secretsDatasetID, secretsTableID)
	_, _ = fmt.Fprintf(&insertQuery, "SELECT '%v', '%v' FROM UNNEST([1]) ", key, encryptionKey.String())
	_, _ = fmt.Fprintf(&insertQuery, "WHERE NOT EXISTS (SELECT 1 FROM `%v.%v.%v` WHERE key = '%v')", a.gcpProject, secretsDatasetID, secretsTableID, key)

	_, err = a.client.QueryAndWait(ctx, a.gcpProject, insertQuery.String())
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *bigQueryAPI) MakeBigQueryUrlForJoinableViews(name, projectID, datasetID, tableID string) string {
	return fmt.Sprintf("%v.%v.%v", a.gcpProject, name, makeJoinableViewName(projectID, datasetID, tableID))
}

func (a *bigQueryAPI) Grant(ctx context.Context, projectID, datasetID, tableID, member string) error {
	const op errs.Op = "bigQueryAPI.Grant"

	entType, entity := strings.Split(member, ":")[0], strings.Split(member, ":")[1]

	var entityType bq.EntityType
	switch entType {
	case "user", "serviceAccount":
		entityType = bq.UserEmailEntity
	case "group":
		entityType = bq.GroupEmailEntity
	}

	entry := &bq.AccessEntry{
		Role:       bq.BigQueryMetadataViewerRole,
		Entity:     entity,
		EntityType: entityType,
	}

	err := a.client.AddDatasetRoleAccessEntry(
		ctx,
		projectID,
		datasetID,
		entry,
	)
	if err != nil {
		return errs.E(errs.IO, op, fmt.Errorf("adding dataset role access entry: %w", err))
	}

	err = a.client.AddAndSetTablePolicy(ctx, projectID, datasetID, tableID, bq.BigQueryDataViewerRole.String(), member)
	if err != nil {
		return errs.E(errs.IO, op, fmt.Errorf("adding and setting table policy: %w", err))
	}

	return nil
}

func (a *bigQueryAPI) Revoke(ctx context.Context, projectID, datasetID, tableID, member string) error {
	const op errs.Op = "bigQueryAPI.Revoke"

	err := a.client.RemoveAndSetTablePolicy(ctx, projectID, datasetID, tableID, bq.BigQueryDataViewerRole.String(), member)
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	// FIXME: should we also remove the access entry

	return nil
}

func (a *bigQueryAPI) AddToAuthorizedViews(ctx context.Context, srcProjectID, srcDataset, sinkProjectID, sinkDataset, sinkTable string) error {
	const op errs.Op = "bigQueryAPI.AddToAuthorizedViews"

	err := a.client.AddDatasetViewAccessEntry(ctx, srcProjectID, srcDataset, &bq.View{
		ProjectID: sinkProjectID,
		DatasetID: sinkDataset,
		TableID:   sinkTable,
	})
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	return nil
}

func NewBigQueryAPI(gcpProject, gcpRegion, pseudoDataSet string, client bq.Operations) *bigQueryAPI {
	return &bigQueryAPI{
		client:        client,
		gcpProject:    gcpProject,
		gcpRegion:     gcpRegion,
		pseudoDataSet: pseudoDataSet,
	}
}
