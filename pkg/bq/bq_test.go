package bq_test

import (
	"context"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type CleanupFn func()

func NewBigQueryEmulator(id string, project server.Source, t *testing.T) (string, CleanupFn) {
	bqServer, err := server.New(server.TempStorage)
	if err != nil {
		t.Fatalf("creating bigquery emulator: %v", err)
	}

	err = bqServer.Load(project)
	if err != nil {
		t.Fatalf("initializing bigquery emulator: %v", err)
	}

	if err := bqServer.SetProject(id); err != nil {
		t.Fatalf("setting project: %v", err)
	}
	testServer := bqServer.TestServer()

	return testServer.URL, func() {
		testServer.Close()
	}
}

func project(id string, datasets ...*types.Dataset) server.Source {
	p := &types.Project{
		ID: id,
	}

	for _, dataset := range datasets {
		p.Datasets = append(p.Datasets, dataset)
	}

	return server.StructSource(p)
}

func dataset(id string, tables ...*types.Table) *types.Dataset {
	return &types.Dataset{
		ID:     id,
		Tables: tables,
	}
}

func table(id string, columns ...*types.Column) *types.Table {
	return &types.Table{
		ID:      id,
		Columns: columns,
	}
}

func columnNullable(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.NullableMode,
	}
}

func columnRequired(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.RequiredMode,
	}
}

func columnRepeated(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.RepeatedMode,
	}
}

func projectWithDatasetAndTable(projectID, datasetID, tableID string, columns ...*types.Column) server.Source {
	return project(projectID, dataset(datasetID, table(tableID, columns...)))
}

func TestClient_GetDataset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
			expect: &bq.Dataset{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
			},
		},
		{
			name:      "not found",
			projectID: "test-project",
			datasetID: "test-dataset",
			project:   project("test-project"),
			expect:    bq.ErrNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.GetDataset(context.Background(), tc.projectID, tc.datasetID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}

func TestClient_GetDatasets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
			expect: []*bq.Dataset{
				{
					ProjectID: "test-project",
					DatasetID: "test-dataset",
				},
			},
			expectErr: false,
		},
		{
			name:      "no datasets",
			projectID: "test-project",
			project:   project("test-project", []*types.Dataset{}...),
			expect:    []*bq.Dataset{},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.GetDatasets(context.Background(), tc.projectID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}

func TestClient_GetTables(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
				columnRequired("test-column-required"),
				columnRepeated("test-column-repeated"),
			),
			expect: []*bq.Table{
				{
					ProjectID:   "test-project",
					DatasetID:   "test-dataset",
					TableID:     "test-table",
					Name:        "",
					Description: "",
					Type:        bq.RegularTable,
					Schema: []*bq.Column{
						{
							Name: "test-column",
							Type: bq.StringFieldType,
							Mode: bq.NullableMode,
						},
						{
							Name: "test-column-required",
							Type: bq.StringFieldType,
							Mode: bq.RequiredMode,
						},
						{
							Name: "test-column-repeated",
							Type: bq.StringFieldType,
							Mode: bq.RepeatedMode,
						},
					},
					LastModified: time.Now(),
					Created:      time.Now(),
					Expires:      time.Now(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.GetTables(context.Background(), tc.projectID, tc.datasetID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				cmp.Equal(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
			}
		})
	}
}

func TestClient_GetTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		tableID   string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
				columnRequired("test-column-required"),
				columnRepeated("test-column-repeated"),
			),
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Type:      bq.RegularTable,
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
					{
						Name: "test-column-required",
						Type: bq.StringFieldType,
						Mode: bq.RequiredMode,
					},
					{
						Name: "test-column-repeated",
						Type: bq.StringFieldType,
						Mode: bq.RepeatedMode,
					},
				},
				LastModified: time.Now(),
				Created:      time.Now(),
				Expires:      time.Now(),
			},
		},
		{
			name:      "not found",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			project:   project("test-project"),
			expect:    bq.ErrNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.GetTable(context.Background(), tc.projectID, tc.datasetID, tc.tableID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, err)
			} else {
				assert.NoError(t, err)
				cmp.Equal(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
			}
		})
	}
}

func TestClient_CreateDataset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			project:   project("test-project", []*types.Dataset{}...),
			expect: []*bq.Dataset{
				{
					ProjectID: "test-project",
					DatasetID: "test-dataset",
				},
			},
			expectErr: false,
		},
		// Emulator doesn't throw a conflict error when creating an existing dataset
		// - PR in the works: https://github.com/goccy/bigquery-emulator/pull/304
		// {
		// 	name:      "already exists",
		// 	projectID: "test-project",
		// 	datasetID: "test-dataset",
		// 	project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
		// 		columnNullable("test-column"),
		// 		columnRequired("test-column-required"),
		// 		columnRepeated("test-column-repeated"),
		// 	),
		// 	expect:    bq.ErrExist,
		// 	expectErr: true,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, project(tc.projectID), t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.CreateDataset(context.Background(), tc.projectID, tc.datasetID, "europe-north1")
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetDatasets(context.Background(), tc.projectID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}

func TestClient_CreateDatasetIfNotExists(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			project:   project("test-project", []*types.Dataset{}...),
			expect: []*bq.Dataset{
				{
					ProjectID: "test-project",
					DatasetID: "test-dataset",
				},
			},
		},
		// Emulator doesn't throw a conflict error when creating an existing dataset
		// - PR in the works: https://github.com/goccy/bigquery-emulator/pull/304
		// {
		// 	name:      "dont fail on already exists",
		// 	projectID: "test-project",
		// 	datasetID: "test-dataset",
		// 	project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
		// 		columnNullable("test-column"),
		// 		columnRequired("test-column-required"),
		// 		columnRepeated("test-column-repeated"),
		// 	),
		// 	expect: []*bq.Dataset{
		// 		{
		// 			ProjectID: "test-project",
		// 			DatasetID: "test-dataset",
		// 		},
		// 	},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.CreateDatasetIfNotExists(context.Background(), tc.projectID, tc.datasetID, "europe-north1")
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetDatasets(context.Background(), tc.projectID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}

func TestDatasetNameWithRandomPostfix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		dataset  string
		expected string
	}{
		{
			name:     "success",
			dataset:  "test-dataset",
			expected: "test_dataset_",
		},
		{
			name:     "starts with number",
			dataset:  "1-test-dataset",
			expected: "A1_test_dataset_",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := bq.DatasetNameWithRandomPostfix(tc.dataset)
			assert.Contains(t, got, tc.expected)
			fmt.Println(got)
		})
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		validator validation.Validatable
		expectErr bool
		expect    any
	}{
		{
			name: "success",
			validator: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
			},
		},
		{
			name:      "missing all required fields",
			validator: &bq.Table{},
			expect:    "DatasetID: cannot be blank; Location: cannot be blank; ProjectID: cannot be blank; TableID: cannot be blank.",
			expectErr: true,
		},
		{
			name: "schema and view are mutually exclusive",
			validator: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
				ViewQuery: "SELECT * FROM table",
			},
			expect:    "Schema: Schema must be empty when ViewQuery is provided; ViewQuery: ViewQuery must be empty when Schema is provided.",
			expectErr: true,
		},
		{
			name: "just view query is fine",
			validator: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				ViewQuery: "SELECT * FROM table",
			},
		},
		{
			name: "just schema is fine",
			validator: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.validator.Validate()
			if tc.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tc.expect, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_CreateTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		table     *bq.Table
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name: "success",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
			},
			project: project("test-project", dataset("test-dataset")),
		},
		{
			name: "already exists",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
			},
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
				columnRequired("test-column-required"),
				columnRepeated("test-column-repeated"),
			),
			expect:    bq.ErrExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.table.ProjectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.CreateTable(context.Background(), tc.table)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetTables(context.Background(), tc.table.ProjectID, tc.table.DatasetID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				cmp.Equal(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
			}
		})
	}
}

func TestClient_CreateTableOrUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		table     *bq.Table
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name: "it works when table doesn't exist",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
			project: project("test-project", dataset("test-dataset")),
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
		},
		{
			name: "it works when table exists and schema is the same",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
		},
		{
			name: "it works when table exists and we update the schema",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
				columnRequired("test-column-to-be-removed"),
			),
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
				Schema: []*bq.Column{
					{
						Name: "test-column",
						Type: bq.StringFieldType,
						Mode: bq.NullableMode,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.table.ProjectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.CreateTableOrUpdate(context.Background(), tc.table)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				cmp.Equal(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
			}
		})
	}
}

func TestClient_DeleteDataset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		projectID     string
		datasetID     string
		deleteContent bool
		project       server.Source
		expectErr     bool
	}{
		{
			name:      "deletes dataset that exists",
			projectID: "test-project",
			datasetID: "test-dataset",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
		},
		{
			name:      "does not error while deleting non-existent dataset",
			projectID: "test-project",
			datasetID: "test-dataset",
			project:   project("test-project"),
		},
		{
			name:      "can also delete using with delete content",
			projectID: "test-project",
			datasetID: "test-dataset",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
			deleteContent: true,
		},
		{
			name:          "does not error while deleting non-existent dataset with delete content",
			projectID:     "test-project",
			datasetID:     "test-dataset",
			project:       project("test-project"),
			deleteContent: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.DeleteDataset(context.Background(), tc.projectID, tc.datasetID, tc.deleteContent)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetDatasets(context.Background(), tc.projectID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Empty(t, got)
			}
		})
	}
}

func TestClient_DeleteTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		tableID   string
		project   server.Source
		expectErr bool
	}{
		{
			name:      "deletes table that exists",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
		},
		{
			name:      "does not error while deleting non-existent table",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			project:   project("test-project", dataset("test-dataset")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.DeleteTable(context.Background(), tc.projectID, tc.datasetID, tc.tableID)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetTables(context.Background(), tc.projectID, tc.datasetID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Empty(t, got)
			}
		})
	}
}

const testData = `projects:
- id: test-project
  datasets:
    - id: test-dataset
      tables:
        - id: test-table
          columns:
            - name: id
              type: INTEGER
            - name: name
              type: STRING
            - name: createdAt
              type: TIMESTAMP
          data:
            - id: 1
              name: alice
              createdAt: "2022-10-21T00:00:00"
            - id: 2
              name: bob
              createdAt: "2022-10-21T00:00:00"`

func TestClient_QueryAndWait(t *testing.T) {
	t.Parallel()

	fileFromYAML := func(t *testing.T, data string) string {
		dir := t.TempDir()

		testFilePath := filepath.Join(dir, "test.yaml")

		err := os.WriteFile(testFilePath, []byte(data), 0644)
		assert.NoError(t, err)

		return testFilePath
	}

	testCases := []struct {
		name      string
		projectID string
		query     string
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "should work",
			projectID: "test-project",
			query:     "SELECT * FROM `test-dataset.test-table`",
			project:   server.YAMLSource(fileFromYAML(t, testData)),
			expect: &bq.JobStatistics{
				TotalBytesProcessed: 34,
			},
			expectErr: false,
		},
		{
			name:      "should fail",
			projectID: "test-project",
			query:     "SELECT * FROM `test-dataset.test-table-nope`",
			project:   server.YAMLSource(fileFromYAML(t, testData)),
			expect:    "waiting for query: googleapi: Error 400: failed to analyze: INVALID_ARGUMENT: Table not found: `test-dataset.test-table-nope`; Did you mean test-dataset.test-table? [at 1:15], jobInternalError",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			got, err := c.QueryAndWait(context.Background(), tc.projectID, tc.query)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				cmp.Equal(tc.expect, got, cmpopts.IgnoreFields(bq.JobStatistics{}, "CreationTime", "StartTime", "EndTime"))
			}
		})
	}
}

// A little bit unsure if this actually does anything behind the scenes
func TestClient_AddDatasetRoleAccessEntry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		project   server.Source
		input     *bq.AccessEntry
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			input: &bq.AccessEntry{
				Role:       bq.ReaderRole,
				Entity:     "bob@example.com",
				EntityType: bq.UserEmailEntity,
			},
			project: project("test-project", dataset("test-dataset")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.AddDatasetRoleAccessEntry(context.Background(), tc.projectID, tc.datasetID, tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// A little bit unsure if this actually does anything behind the scenes
func TestClient_AddDatasetViewAccessEntry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		input     *bq.View
		project   server.Source
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			input: &bq.View{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
			},
			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
				columnNullable("test-column"),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
			defer cleanup()

			c := bq.NewClient(url, false)

			err := c.AddDatasetViewAccessEntry(context.Background(), tc.projectID, tc.datasetID, tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// IAM endpoints are not implemented, so these tests will fail
// func TestClient_AddAndSetTablePolicy(t *testing.T) {
// 	t.Parallel()
//
// 	testCases := []struct {
// 		name      string
// 		projectID string
// 		datasetID string
// 		tableID   string
// 		role      string
// 		member    string
// 		project   server.Source
// 	}{
// 		{
// 			name: "success",
// 			project: projectWithDatasetAndTable("test-project", "test-dataset", "test-table",
// 				columnNullable("test-column"),
// 			),
// 			projectID: "test-project",
// 			datasetID: "test-dataset",
// 			tableID:   "test-table",
// 			role:      bq.BigQueryDataViewerRole.String(),
// 			member:    "bob@example.com",
// 		},
// 	}
//
// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			t.Parallel()
//
// 			url, cleanup := NewBigQueryEmulator(tc.projectID, tc.project, t)
// 			defer cleanup()
//
// 			c := bq.NewClient(url, false)
//
// 			ctx := context.Background()
// 			ctx, _ = context.WithDeadline(ctx, time.Now().Add(1*time.Second))
//
// 			err := c.AddAndSetTablePolicy(ctx, tc.projectID, tc.datasetID, tc.tableID, tc.role, tc.member)
// 			assert.NoError(t, err)
// 		})
// 	}
// }
