package bq_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/iam/apiv1/iampb"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/bq/emulator"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestClient_GetDataset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		datasetID string
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
			expect: &bq.Dataset{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
			},
		},
		{
			name:      "not found",
			projectID: "test-project",
			datasetID: "test-dataset",
			expect:    bq.ErrNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
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
			expect:    []*bq.Dataset{},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
					emulator.ColumnRequired("test-column-required"),
					emulator.ColumnRepeated("test-column-repeated"),
				},
			},
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

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			got, err := c.GetTables(context.Background(), tc.projectID, tc.datasetID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				diff := cmp.Diff(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
				assert.Empty(t, diff)
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
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
					emulator.ColumnRequired("test-column-required"),
					emulator.ColumnRepeated("test-column-repeated"),
				},
			},
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
			expect:    bq.ErrNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			got, err := c.GetTable(context.Background(), tc.projectID, tc.datasetID, tc.tableID)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, err)
			} else {
				assert.NoError(t, err)
				diff := cmp.Diff(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
				assert.Empty(t, diff)
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
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
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

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			datasetID: "test-dataset",
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

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		projectID string
		table     *bq.Table
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "success",
			projectID: "test-project",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
			},
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
			},
			expect: []*bq.Table{
				{
					ProjectID: "test-project",
					DatasetID: "test-dataset",
					TableID:   "test-table",
					Type:      bq.RegularTable,
				},
			},
		},
		{
			name:      "already exists",
			projectID: "test-project",
			table: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Location:  "europe-north1",
			},
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
					emulator.ColumnRequired("test-column-required"),
					emulator.ColumnRepeated("test-column-repeated"),
				},
			},
			expect:    bq.ErrExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			err := c.CreateTable(context.Background(), tc.table)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := c.GetTables(context.Background(), tc.table.ProjectID, tc.table.DatasetID)
				assert.NoError(t, err)
				assert.NotNil(t, got)
				diff := cmp.Diff(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
				assert.Empty(t, diff)
			}
		})
	}
}

func TestClient_CreateTableOrUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		projectID string
		table     *bq.Table
		schema    *emulator.Dataset
		expect    any
		expectErr bool
	}{
		{
			name:      "it works when table doesn't exist",
			projectID: "test-project",
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
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
			},
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				// bq-emulator ignores storing location so it's ""
				Location: "",
				Type:     bq.RegularTable,
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
			name:      "it works when table exists and schema is the same",
			projectID: "test-project",
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
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				// We can't change the location after creating
				// and it was ""
				Location: "",
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
			name:      "it works when table exists and we update the schema",
			projectID: "test-project",
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
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
					emulator.ColumnRequired("test-column-to-be-removed"),
				},
			},
			expect: &bq.Table{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				// We can't change the location after creating
				// and it was ""
				Location: "",
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

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			got, err := c.CreateTableOrUpdate(context.Background(), tc.table)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				diff := cmp.Diff(
					tc.expect,
					got,
					cmpopts.IgnoreFields(bq.Table{}, "LastModified", "Created", "Expires"),
					cmpopts.IgnoreUnexported(bq.Table{}),
				)
				assert.Empty(t, diff)
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
		schema        *emulator.Dataset
		expectErr     bool
	}{
		{
			name:      "deletes dataset that exists",
			projectID: "test-project",
			datasetID: "test-dataset",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
		},
		{
			name:      "does not error while deleting non-existent dataset",
			projectID: "test-project",
			datasetID: "test-dataset",
		},
		{
			name:      "can also delete using with delete content",
			projectID: "test-project",
			datasetID: "test-dataset",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
			deleteContent: true,
		},
		{
			name:          "does not error while deleting non-existent dataset with delete content",
			projectID:     "test-project",
			datasetID:     "test-dataset",
			deleteContent: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		schema    *emulator.Dataset
		expectErr bool
	}{
		{
			name:      "deletes table that exists",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
		},
		{
			name:      "does not error while deleting non-existent table",
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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

		err := os.WriteFile(testFilePath, []byte(data), 0o644)
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

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithSource(tc.projectID, tc.project)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			got, err := c.QueryAndWait(context.Background(), tc.projectID, tc.query)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tc.expect, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				diff := cmp.Diff(tc.expect, got, cmpopts.IgnoreFields(bq.JobStatistics{}, "CreationTime", "StartTime", "EndTime"))
				assert.Empty(t, diff)
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
		schema    *emulator.Dataset
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
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

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
		schema    *emulator.Dataset
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
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)
			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			err := c.AddDatasetViewAccessEntry(context.Background(), tc.projectID, tc.datasetID, tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// IAM endpoints are not implemented, so we need to mock them
func TestClient_AddAndSetTablePolicy(t *testing.T) {
	testCases := []struct {
		name          string
		projectID     string
		datasetID     string
		tableID       string
		role          string
		member        string
		schema        *emulator.Dataset
		currentPolicy *iampb.Policy
		expect        any
		expectErr     bool
	}{
		{
			name: "success",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			role:      bq.BigQueryDataViewerRole.String(),
			member:    "bob@example.com",
			currentPolicy: &iampb.Policy{
				Version: 1,
			},
			expect: &iampb.Policy{
				Version: 1,
				Bindings: []*iampb.Binding{
					{
						Role: bq.BigQueryDataViewerRole.String(),
						Members: []string{
							"bob@example.com",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)

			log := zerolog.New(os.Stdout)

			got := &iampb.SetIamPolicyRequest{}
			// We need to enable the mock interceptor, since the IAM endpoints are not implemented
			s.EnableMock(false, log,
				emulator.DatasetTableIAMPolicyGetMock(tc.projectID, tc.datasetID, tc.tableID, log, tc.currentPolicy),
				emulator.DatasetTableIAMPolicySetMock(tc.projectID, tc.datasetID, tc.tableID, log, got),
			)

			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			ctx := context.Background()
			ctx, _ = context.WithDeadline(ctx, time.Now().Add(1*time.Second))

			err := c.AddAndSetTablePolicy(ctx, tc.projectID, tc.datasetID, tc.tableID, tc.role, tc.member)
			assert.NoError(t, err)
			diff := cmp.Diff(tc.expect, got.Policy, cmpopts.IgnoreUnexported(iampb.Policy{}), cmpopts.IgnoreUnexported(iampb.Binding{}))
			assert.Empty(t, diff)
		})
	}
}

func TestClient_RemoveAndSetTablePolicy(t *testing.T) {
	testCases := []struct {
		name          string
		projectID     string
		datasetID     string
		tableID       string
		role          string
		member        string
		schema        *emulator.Dataset
		currentPolicy *iampb.Policy
		expect        any
		expectErr     bool
	}{
		{
			name: "success",
			schema: &emulator.Dataset{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Columns: []*types.Column{
					emulator.ColumnNullable("test-column"),
				},
			},
			projectID: "test-project",
			datasetID: "test-dataset",
			tableID:   "test-table",
			role:      bq.BigQueryDataViewerRole.String(),
			member:    "bob@example.com",
			currentPolicy: &iampb.Policy{
				Version: 1,
				Bindings: []*iampb.Binding{
					{
						Role: bq.BigQueryDataViewerRole.String(),
						Members: []string{
							"bob@example.com",
						},
					},
				},
			},
			expect: &iampb.Policy{Version: 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := emulator.New(zerolog.New(os.Stdout))
			defer s.Cleanup()

			s.WithProject(tc.projectID, tc.schema)

			got := &iampb.SetIamPolicyRequest{}

			log := zerolog.New(os.Stdout)

			// We need to enable the mock interceptor, since the IAM endpoints are not implemented
			s.EnableMock(false, log,
				emulator.DatasetTableIAMPolicyGetMock(tc.projectID, tc.datasetID, tc.tableID, log, tc.currentPolicy),
				emulator.DatasetTableIAMPolicySetMock(tc.projectID, tc.datasetID, tc.tableID, log, got),
			)

			s.TestServer()

			c := bq.NewClient(s.Endpoint(), false, zerolog.Nop())

			ctx := context.Background()
			ctx, _ = context.WithDeadline(ctx, time.Now().Add(1*time.Second))

			err := c.RemoveAndSetTablePolicy(ctx, tc.projectID, tc.datasetID, tc.tableID, tc.role, tc.member)
			assert.NoError(t, err)
			diff := cmp.Diff(tc.expect, got.Policy, cmpopts.IgnoreUnexported(iampb.Policy{}), cmpopts.IgnoreUnexported(iampb.Binding{}))
			assert.Empty(t, diff)
		})
	}
}
