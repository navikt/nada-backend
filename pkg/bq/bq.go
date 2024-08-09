package bq

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/rs/zerolog"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/iam"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/lithammer/shortuuid/v4"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var _ Operations = &Client{}

type Operations interface {
	GetDataset(ctx context.Context, projectID, datasetID string) (*Dataset, error)
	GetDatasets(ctx context.Context, projectID string) ([]*Dataset, error)
	GetTable(ctx context.Context, projectID, datasetID, tableID string) (*Table, error)
	GetTables(ctx context.Context, projectID, datasetID string) ([]*Table, error)
	CreateDataset(ctx context.Context, projectID, datasetID, region string) error
	CreateDatasetIfNotExists(ctx context.Context, projectID, datasetID, region string) error
	CreateTable(ctx context.Context, input *Table) error
	CreateTableOrUpdate(ctx context.Context, input *Table) (*Table, error)
	DeleteTable(ctx context.Context, projectID, datasetID, tableID string) error
	DeleteDataset(ctx context.Context, projectID, datasetID string, deleteContents bool) error
	QueryAndWait(ctx context.Context, projectID, query string) (*JobStatistics, error)
	AddDatasetRoleAccessEntry(ctx context.Context, projectID, datasetID string, input *AccessEntry) error
	AddDatasetViewAccessEntry(ctx context.Context, projectID, datasetID string, view *View) error
	AddAndSetTablePolicy(ctx context.Context, projectID, datasetID, tableID, role, member string) error
	RemoveAndSetTablePolicy(ctx context.Context, projectID, datasetID, tableID, role, member string) error
}

var (
	ErrExist    = errors.New("already exists")
	ErrNotExist = errors.New("not exists")
)

type Client struct {
	endpoint             string
	enableAuthentication bool
	log                  zerolog.Logger
}

type TableType string

func (t *TableType) String() string {
	if t == nil {
		return ""
	}

	return string(*t)
}

const (
	// RegularTable is a regular table.
	RegularTable TableType = "TABLE"
	// ViewTable is a table type describing that the table is a logical view.
	// See more information at https://cloud.google.com//docs/views.
	ViewTable TableType = "VIEW"
	// ExternalTable is a table type describing that the table is an external
	// table (also known as a federated data source). See more information at
	// https://cloud.google.com/bigquery/external-data-sources.
	ExternalTable TableType = "EXTERNAL"
	// MaterializedView represents a managed storage table that's derived from
	// a base table.
	MaterializedView TableType = "MATERIALIZED_VIEW"
	// Snapshot represents an immutable point in time snapshot of some other
	// table.
	Snapshot TableType = "SNAPSHOT"
)

type ColumnMode string

func (c *ColumnMode) String() string {
	if c == nil {
		return ""
	}

	return string(*c)
}

const (
	NullableMode ColumnMode = "NULLABLE"
	RequiredMode ColumnMode = "REQUIRED"
	RepeatedMode ColumnMode = "REPEATED"
)

type FieldType string

func (f *FieldType) String() string {
	if f == nil {
		return ""
	}

	return string(*f)
}

const (
	// StringFieldType is a string field type.
	StringFieldType FieldType = "STRING"
	// BytesFieldType is a bytes field type.
	BytesFieldType FieldType = "BYTES"
	// IntegerFieldType is a integer field type.
	IntegerFieldType FieldType = "INTEGER"
	// FloatFieldType is a float field type.
	FloatFieldType FieldType = "FLOAT"
	// BooleanFieldType is a boolean field type.
	BooleanFieldType FieldType = "BOOLEAN"
	// TimestampFieldType is a timestamp field type.
	TimestampFieldType FieldType = "TIMESTAMP"
	// RecordFieldType is a record field type. It is typically used to create columns with repeated or nested data.
	RecordFieldType FieldType = "RECORD"
	// DateFieldType is a date field type.
	DateFieldType FieldType = "DATE"
	// TimeFieldType is a time field type.
	TimeFieldType FieldType = "TIME"
	// DateTimeFieldType is a datetime field type.
	DateTimeFieldType FieldType = "DATETIME"
	// NumericFieldType is a numeric field type. Numeric types include integer types, floating point types and the
	// NUMERIC data type.
	NumericFieldType FieldType = "NUMERIC"
	// GeographyFieldType is a string field type.  Geography types represent a set of points
	// on the Earth's surface, represented in Well Known Text (WKT) format.
	GeographyFieldType FieldType = "GEOGRAPHY"
	// BigNumericFieldType is a numeric field type that supports values of larger precision
	// and scale than the NumericFieldType.
	BigNumericFieldType FieldType = "BIGNUMERIC"
	// IntervalFieldType is a representation of a duration or an amount of time.
	IntervalFieldType FieldType = "INTERVAL"
	// JSONFieldType is a representation of a json object.
	JSONFieldType FieldType = "JSON"
	// RangeFieldType represents a continuous range of values.
	RangeFieldType FieldType = "RANGE"
)

// AccessRole is the level of access to grant to a dataset.
type AccessRole string

const (
	// OwnerRole is the OWNER AccessRole.
	OwnerRole AccessRole = "OWNER"
	// ReaderRole is the READER AccessRole.
	ReaderRole AccessRole = "READER"
	// WriterRole is the WRITER AccessRole.
	WriterRole AccessRole = "WRITER"

	BigQueryMetadataViewerRole AccessRole = "roles/bigquery.metadataViewer"
	BigQueryDataViewerRole     AccessRole = "roles/bigquery.dataViewer"
)

func (r AccessRole) String() string {
	return string(r)
}

type EntityType string

const (
	UserEmailEntity  EntityType = "user"
	GroupEmailEntity EntityType = "group"
	ViewEntity       EntityType = "view"
)

func (e EntityType) String() string {
	return string(e)
}

type Dataset struct {
	ProjectID string
	DatasetID string

	Name        string
	Description string

	Access []*AccessEntry
}

type AccessEntry struct {
	Role       AccessRole
	EntityType EntityType
	Entity     string
	View       *View
}

func (a AccessEntry) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.EntityType,
			validation.Required,
			validation.In(
				UserEmailEntity,
				GroupEmailEntity,
				ViewEntity,
			),
		),
		validation.Field(&a.Role,
			validation.Nil.When(a.EntityType == ViewEntity).Error("Role must be nil when EntityType is ViewEntity"),
			validation.Required.When(a.EntityType != ViewEntity).Error("Role is required when EntityType is not ViewEntity"),
		),
		validation.Field(&a.Entity,
			validation.Nil.When(a.EntityType == ViewEntity).Error("Entity must be nil when EntityType is ViewEntity"),
			validation.Required.When(a.EntityType != ViewEntity).Error("Entity is required when EntityType is not ViewEntity"),
		),
		// When EntityType: ViewEntity
		validation.Field(&a.View,
			validation.Nil.When(a.EntityType != ViewEntity).Error("View must be nil when EntityType is not ViewEntity"),
			validation.Required.When(a.EntityType == ViewEntity).Error("View is required when EntityType is ViewEntity"),
		),
	)
}

type View struct {
	ProjectID string
	DatasetID string
	TableID   string
}

func (v View) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ProjectID, validation.Required),
		validation.Field(&v.DatasetID, validation.Required),
		validation.Field(&v.TableID, validation.Required),
	)
}

type Table struct {
	ProjectID string
	DatasetID string
	TableID   string
	Location  string

	Name        string
	Description string
	Type        TableType

	// The table schema. If provided on create, ViewQuery must be empty.
	Schema []*Column

	// The query to use for a logical view. If provided on create, Schema must be nil.
	ViewQuery string

	LastModified time.Time
	Created      time.Time
	Expires      time.Time

	etag string
}

func (t Table) Validate() error {
	return validation.ValidateStruct(&t,
		validation.Field(&t.ProjectID, validation.Required),
		validation.Field(&t.DatasetID, validation.Required),
		validation.Field(&t.TableID, validation.Required),
		validation.Field(&t.Location, validation.Required, validation.In("europe-north1")),
		validation.Field(&t.Schema, validation.Nil.When(t.ViewQuery != "").Error("Schema must be empty when ViewQuery is provided")),
		validation.Field(&t.ViewQuery, validation.Empty.When(t.Schema != nil).Error("ViewQuery must be empty when Schema is provided")),
	)
}

type Column struct {
	Name        string
	Type        FieldType
	Mode        ColumnMode
	Description string
}

func (c Column) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.Type, validation.Required),
		validation.Field(&c.Mode, validation.Required),
	)
}

type JobStatistics struct {
	CreationTime        time.Time
	StartTime           time.Time
	EndTime             time.Time
	TotalBytesProcessed int64
}

func (c *Client) GetDataset(ctx context.Context, projectID, datasetID string) (*Dataset, error) {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("dataset: %w", err)
	}

	dataset, err := c.getDatasetWithMetadata(ctx, client, datasetID)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func (c *Client) getDatasetWithMetadata(ctx context.Context, client *bigquery.Client, datasetID string) (*Dataset, error) {
	meta, err := client.Dataset(datasetID).Metadata(ctx)
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusNotFound {
			return nil, ErrNotExist
		}

		return nil, fmt.Errorf("getting dataset metadata %s: %w", datasetID, err)
	}

	var access []*AccessEntry
	for _, a := range meta.Access {
		var entityType EntityType
		switch a.EntityType {
		case bigquery.UserEmailEntity:
			entityType = UserEmailEntity
		case bigquery.GroupEmailEntity:
			entityType = GroupEmailEntity
		case bigquery.ViewEntity:
			entityType = ViewEntity
		default:
			return nil, fmt.Errorf("unknown entity type %v", a.EntityType)
		}

		var view *View
		if a.View != nil {
			view = &View{
				ProjectID: a.View.ProjectID,
				DatasetID: a.View.DatasetID,
				TableID:   a.View.TableID,
			}
		}

		access = append(access, &AccessEntry{
			Role:       AccessRole(a.Role),
			EntityType: entityType,
			Entity:     a.Entity,
			View:       view,
		})
	}

	return &Dataset{
		ProjectID:   client.Project(),
		DatasetID:   datasetID,
		Name:        meta.Name,
		Description: meta.Description,
		Access:      access,
	}, nil
}

func (c *Client) GetDatasets(ctx context.Context, projectID string) ([]*Dataset, error) {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("datasets: %w", err)
	}

	datasets := []*Dataset{}
	it := client.Datasets(ctx)
	for {
		ds, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			return nil, fmt.Errorf("iterating datasets: %w", err)
		}

		datasets = append(datasets, &Dataset{
			ProjectID: ds.ProjectID,
			DatasetID: ds.DatasetID,
		})
	}

	return datasets, nil
}

func (c *Client) GetTables(ctx context.Context, projectID, datasetID string) ([]*Table, error) {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("tables: %w", err)
	}

	tables := []*Table{}
	it := client.Dataset(datasetID).Tables(ctx)
	for {
		t, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			return nil, fmt.Errorf("iterating tables: %w", err)
		}

		table, err := c.getTableWithMetadata(ctx, client, datasetID, t.TableID)
		if err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func (c *Client) GetTable(ctx context.Context, projectID, datasetID, tableID string) (*Table, error) {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("table: %w", err)
	}

	table, err := c.getTableWithMetadata(ctx, client, datasetID, tableID)
	if err != nil {
		return nil, err
	}

	return table, nil
}

func schemaToFieldSchema(schema []*Column) []*bigquery.FieldSchema {
	if len(schema) == 0 {
		return nil
	}

	fields := make([]*bigquery.FieldSchema, len(schema))
	for i, col := range schema {
		f := &bigquery.FieldSchema{
			Name:        col.Name,
			Type:        bigquery.FieldType(col.Type),
			Description: col.Description,
		}

		if col.Mode == RepeatedMode {
			f.Repeated = true
		}

		if col.Mode == RequiredMode {
			f.Required = true
		}

		fields[i] = f
	}

	return fields
}

func fieldSchemaToSchema(fields []*bigquery.FieldSchema) []*Column {
	if len(fields) == 0 {
		return nil
	}

	schema := make([]*Column, len(fields))

	for i, f := range fields {
		mode := NullableMode

		if f.Repeated {
			mode = RepeatedMode
		}

		if f.Required && !f.Repeated {
			mode = RequiredMode
		}

		schema[i] = &Column{
			Name:        f.Name,
			Type:        FieldType(f.Type),
			Mode:        mode,
			Description: f.Description,
		}
	}

	return schema
}

func (c *Client) getTableWithMetadata(ctx context.Context, client *bigquery.Client, datasetID, tableID string) (*Table, error) {
	meta, err := client.Dataset(datasetID).Table(tableID).Metadata(ctx)
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusNotFound {
			return nil, ErrNotExist
		}

		return nil, fmt.Errorf("getting table metadata %s.%s.%s: %w", client.Project(), datasetID, tableID, err)
	}

	return &Table{
		ProjectID:    client.Project(),
		DatasetID:    datasetID,
		TableID:      tableID,
		Name:         meta.Name,
		Description:  meta.Description,
		Type:         TableType(meta.Type), // Don't bother with checking validity, coming from the API.
		Schema:       fieldSchemaToSchema(meta.Schema),
		LastModified: meta.LastModifiedTime,
		Created:      meta.CreationTime,
		Expires:      meta.ExpirationTime,
		etag:         meta.ETag,
	}, nil
}

func (c *Client) CreateDataset(ctx context.Context, projectID, datasetID, region string) error {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("creating dataset: %w", err)
	}

	meta := &bigquery.DatasetMetadata{
		Location: region,
	}

	err = client.Dataset(datasetID).Create(ctx, meta)
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusConflict {
			return ErrExist
		}

		return fmt.Errorf("creating dataset %s.%s: %w", projectID, datasetID, err)
	}

	return nil
}

func (c *Client) CreateDatasetIfNotExists(ctx context.Context, projectID, datasetID, region string) error {
	err := c.CreateDataset(ctx, projectID, datasetID, region)
	if errors.Is(err, ErrExist) {
		return nil
	}

	return err
}

func (c *Client) CreateTable(ctx context.Context, input *Table) error {
	err := input.Validate()
	if err != nil {
		return fmt.Errorf("validating table: %w", err)
	}

	client, err := c.clientFromProject(ctx, input.ProjectID)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	err = client.Dataset(input.DatasetID).Table(input.TableID).Create(ctx, &bigquery.TableMetadata{
		Name:        input.Name,
		Location:    input.Location,
		Description: input.Description,
		ViewQuery:   input.ViewQuery,
		Schema:      schemaToFieldSchema(input.Schema),
	})
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusConflict {
			return ErrExist
		}

		return fmt.Errorf("creating table %s.%s.%s: %w", input.ProjectID, input.DatasetID, input.TableID, err)
	}

	return nil
}

func (c *Client) updateTable(ctx context.Context, input *Table, etag string) (*Table, error) {
	client, err := c.clientFromProject(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("updating table: %w", err)
	}

	update, err := client.Dataset(input.DatasetID).Table(input.TableID).Update(ctx, bigquery.TableMetadataToUpdate{
		Name:           input.Name,
		Description:    input.Description,
		Schema:         schemaToFieldSchema(input.Schema),
		ExpirationTime: input.Expires,
		ViewQuery:      input.ViewQuery,
	}, etag)
	if err != nil {
		return nil, err
	}

	return &Table{
		ProjectID:    input.ProjectID,
		DatasetID:    input.DatasetID,
		TableID:      input.TableID,
		Name:         update.Name,
		Description:  update.Description,
		Type:         TableType(update.Type), // Don't bother with checking validity, coming from the API.
		Schema:       fieldSchemaToSchema(update.Schema),
		LastModified: update.LastModifiedTime,
		Created:      update.CreationTime,
		Expires:      update.ExpirationTime,
		etag:         update.ETag,
	}, nil
}

func (c *Client) CreateTableOrUpdate(ctx context.Context, input *Table) (*Table, error) {
	err := input.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating table: %w", err)
	}

	table, err := c.GetTable(ctx, input.ProjectID, input.DatasetID, input.TableID)
	if err != nil && !errors.Is(err, ErrNotExist) {
		return nil, err
	}

	if errors.Is(err, ErrNotExist) {
		err := c.CreateTable(ctx, input)
		if err != nil {
			return nil, err
		}

		return c.GetTable(ctx, input.ProjectID, input.DatasetID, input.TableID)
	}

	return c.updateTable(ctx, input, table.etag)
}

func (c *Client) DeleteTable(ctx context.Context, project, datasetID, tableID string) error {
	client, err := c.clientFromProject(ctx, project)
	if err != nil {
		return fmt.Errorf("deleting table: %w", err)
	}

	err = client.Dataset(datasetID).Table(tableID).Delete(ctx)
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("deleting table %s.%s.%s: %w", project, datasetID, tableID, err)
	}

	return nil
}

func (c *Client) DeleteDataset(ctx context.Context, project, datasetID string, deleteContents bool) error {
	var err error

	client, err := c.clientFromProject(ctx, project)
	if err != nil {
		return fmt.Errorf("deleting dataset: %w", err)
	}

	if deleteContents {
		err = client.Dataset(datasetID).DeleteWithContents(ctx)
	} else {
		err = client.Dataset(datasetID).Delete(ctx)
	}

	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("deleting dataset %s.%s: %w", project, datasetID, err)
	}

	return nil
}

// Generate unique name for dataset:
// - https://cloud.google.com/bigquery/docs/datasets#dataset-naming
func DatasetNameWithRandomPostfix(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	// Replace disallowed characters with underscore
	cleanName := re.ReplaceAllString(name, "_")

	if len(cleanName) > 0 && (cleanName[0] == '_' || ('0' <= cleanName[0] && cleanName[0] <= '9')) {
		cleanName = "A" + cleanName
	}

	postfix := shortuuid.New()

	fullLengthName := cleanName + "_" + postfix

	if len(fullLengthName) > 1024 {
		// If the resulting string is too long, truncate and keep the UUID
		// This ensures a unique name is maintained
		allowedLength := 1024 - len(postfix) - 1 // -1 for the underscore
		cleanName = cleanName[:allowedLength]
		fullLengthName = cleanName + "_" + postfix
	}

	return fullLengthName
}

func (c *Client) QueryAndWait(ctx context.Context, projectID, query string) (*JobStatistics, error) {
	client, err := c.clientFromProject(context.Background(), projectID)
	if err != nil {
		return nil, fmt.Errorf("query and wait: %w", err)
	}

	job, err := client.Query(query).Run(context.Background())
	if err != nil {
		return nil, fmt.Errorf("running query: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for query: %w", err)
	}

	err = status.Err()
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var stats *JobStatistics
	if status.Statistics != nil {
		stats = &JobStatistics{
			CreationTime:        status.Statistics.CreationTime,
			StartTime:           status.Statistics.StartTime,
			EndTime:             status.Statistics.EndTime,
			TotalBytesProcessed: status.Statistics.TotalBytesProcessed,
		}
	}

	return stats, nil
}

func (c *Client) AddDatasetRoleAccessEntry(ctx context.Context, projectID, datasetID string, input *AccessEntry) error {
	err := input.Validate()
	if err != nil {
		return fmt.Errorf("validating role access entry: %w", err)
	}

	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("adding access entry: %w", err)
	}

	var entityType bigquery.EntityType
	switch input.EntityType {
	case UserEmailEntity:
		entityType = bigquery.UserEmailEntity
	case GroupEmailEntity:
		entityType = bigquery.GroupEmailEntity
	case ViewEntity:
		return fmt.Errorf("view entity should be added via own function")
	default:
		return fmt.Errorf("unknown entity type %v", entityType)
	}

	access := &bigquery.AccessEntry{
		Role:       bigquery.AccessRole(input.Role),
		EntityType: entityType,
		Entity:     input.Entity,
	}

	c.log.Info().Fields(map[string]interface{}{
		"project":     projectID,
		"dataset":     datasetID,
		"entity_type": entityType,
		"entity":      input.Entity,
		"role":        string(input.Role),
	}).Msg("adding_access")

	// FIXME: should we check if the access entry already exists?

	ds := client.Dataset(datasetID)

	meta, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("getting dataset metadata %s.%s: %w", projectID, datasetID, err)
	}

	for _, a := range meta.Access {
		if a.EntityType == entityType && a.Entity == input.Entity && a.Role == bigquery.AccessRole(input.Role) {
			return nil
		}
	}

	for _, a := range meta.Access {
		c.log.Info().Fields(map[string]interface{}{
			"project":     projectID,
			"dataset":     datasetID,
			"entity_type": a.EntityType,
			"entity":      a.Entity,
			"role":        string(a.Role),
		}).Msg("existing_access")
	}

	_, err = ds.Update(ctx, bigquery.DatasetMetadataToUpdate{
		Access: append(meta.Access, access),
	}, meta.ETag)
	if err != nil {
		return fmt.Errorf("updating dataset metadata %s.%s: %w", projectID, datasetID, err)
	}

	return nil
}

func (c *Client) AddDatasetViewAccessEntry(ctx context.Context, projectID, datasetID string, input *View) error {
	err := input.Validate()
	if err != nil {
		return fmt.Errorf("validating view access entry: %w", err)
	}

	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("adding view access entry: %w", err)
	}

	access := &bigquery.AccessEntry{
		EntityType: bigquery.ViewEntity,
		View: &bigquery.Table{
			ProjectID: input.ProjectID,
			DatasetID: input.DatasetID,
			TableID:   input.TableID,
		},
	}

	ds := client.Dataset(datasetID)

	meta, err := ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("getting dataset metadata %s: %w", datasetID, err)
	}

	for _, a := range meta.Access {
		if a.EntityType == bigquery.ViewEntity {
			if a.View.ProjectID == input.ProjectID && a.View.DatasetID == input.DatasetID && a.View.TableID == input.TableID {
				return nil
			}
		}
	}

	_, err = ds.Update(ctx, bigquery.DatasetMetadataToUpdate{
		Access: append(meta.Access, access),
	}, meta.ETag)
	if err != nil {
		return fmt.Errorf("updating dataset %s: %w", datasetID, err)
	}

	return nil
}

func (c *Client) AddAndSetTablePolicy(ctx context.Context, projectID, datasetID, tableID, role, member string) error {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("adding table policy: %w", err)
	}

	policy, err := client.Dataset(datasetID).Table(tableID).IAM().Policy(ctx)
	if err != nil {
		return fmt.Errorf("getting table policy: %w", err)
	}

	policy.Add(member, iam.RoleName(role))

	err = client.Dataset(datasetID).Table(tableID).IAM().SetPolicy(ctx, policy)
	if err != nil {
		return fmt.Errorf("setting table policy: %w", err)
	}

	return nil
}

func (c *Client) RemoveAndSetTablePolicy(ctx context.Context, projectID, datasetID, tableID, role, member string) error {
	client, err := c.clientFromProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("removing table policy: %w", err)
	}

	policy, err := client.Dataset(datasetID).Table(tableID).IAM().Policy(ctx)
	if err != nil {
		return fmt.Errorf("getting table policy: %w", err)
	}

	policy.Remove(member, iam.RoleName(role))

	err = client.Dataset(datasetID).Table(tableID).IAM().SetPolicy(ctx, policy)
	if err != nil {
		return fmt.Errorf("setting table policy: %w", err)
	}

	return nil
}

func (c *Client) clientFromProject(ctx context.Context, project string) (*bigquery.Client, error) {
	var options []option.ClientOption

	if c.endpoint != "" {
		options = append(options, option.WithEndpoint(c.endpoint))
	}

	if !c.enableAuthentication {
		options = append(options, option.WithoutAuthentication())
	}

	client, err := bigquery.NewClient(ctx, project, options...)
	if err != nil {
		return nil, fmt.Errorf("creating bigquery client for project %s: %w", project, err)
	}

	return client, nil
}

func NewClient(endpoint string, enableAuthentication bool, log zerolog.Logger) *Client {
	return &Client{
		endpoint:             endpoint,
		enableAuthentication: enableAuthentication,
		log:                  log,
	}
}
