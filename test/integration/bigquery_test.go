package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/goccy/bigquery-emulator/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/bq"
	"github.com/navikt/nada-backend/pkg/bq/emulator"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBigQuery(t *testing.T) {
	log := zerolog.New(os.Stdout)
	c := NewContainers(t, log)
	defer c.Cleanup()

	pgCfg := c.RunPostgres(NewPostgresConfig())

	repo, err := database.New(
		pgCfg.ConnectionURL(),
		10,
		10,
	)
	assert.NoError(t, err)

	gcpProject := "test-project"
	gcpLocation := "europe-north1"
	datasets := []*emulator.Dataset{
		{
			DatasetID: "test-dataset",
			TableID:   "test-table",
			Columns: []*types.Column{
				emulator.ColumnRequired("id"),
				emulator.ColumnNullable("name"),
				emulator.ColumnNullable("description"),
			},
		},
		{
			DatasetID: "pseudo-test-dataset",
		},
	}

	em := emulator.New(log)

	em.WithProject(gcpProject, datasets...)
	em.TestServer()
	bqClient := bq.NewClient(em.Endpoint(), false, zerolog.Nop())

	stores := storage.NewStores(repo, config.Config{}, log)

	zlog := zerolog.New(os.Stdout)
	r := TestRouter(zlog)

	{
		a := gcp.NewBigQueryAPI(gcpProject, gcpLocation, "pseudo-test-dataset", bqClient)
		s := core.NewBigQueryService(stores.BigQueryStorage, a, stores.DataProductsStorage)
		h := handlers.NewBigQueryHandler(s)
		e := routes.NewBigQueryEndpoints(zlog, h)
		f := routes.NewBigQueryRoutes(e)

		// Register routes
		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("Get datasets", func(t *testing.T) {
		expect := &service.BQDatasets{
			BQDatasets: []string{
				"test-dataset",
				"pseudo-test-dataset",
			},
		}
		NewTester(t, server).Get("/api/bigquery/datasets", "projectId", gcpProject).
			HasStatusCode(http.StatusOK).
			Expect(expect, &service.BQDatasets{})
	})

	t.Run("Get tables", func(t *testing.T) {
		expect := &service.BQTables{
			BQTables: []*service.BigQueryTable{
				{
					Name: "test-table",
					Type: "TABLE",
				},
			},
		}
		NewTester(t, server).Get("/api/bigquery/tables", "projectId", gcpProject, "datasetId", "test-dataset").
			HasStatusCode(http.StatusOK).
			Expect(expect, &service.BQTables{}, cmpopts.IgnoreFields(service.BigQueryTable{}, "LastModified"))
	})

	t.Run("Get columns", func(t *testing.T) {
		expect := &service.BQColumns{
			BQColumns: []*service.BigqueryColumn{
				{
					Name: "id",
					Type: "STRING",
					Mode: "REQUIRED",
				},
				{
					Name: "name",
					Type: "STRING",
					Mode: "NULLABLE",
				},
				{
					Name: "description",
					Type: "STRING",
					Mode: "NULLABLE",
				},
			},
		}
		NewTester(t, server).Get("/api/bigquery/columns", "projectId", gcpProject, "datasetId", "test-dataset", "tableId", "test-table").
			HasStatusCode(http.StatusOK).
			Expect(expect, &service.BQColumns{})
	})

	t.Run("Sync tables", func(t *testing.T) {
		user := &service.User{
			Email: "nada@nav.no",
		}

		dp, err := stores.DataProductsStorage.CreateDataproduct(context.Background(), service.NewDataproduct{
			Name:  "My Data Product",
			Group: "nada@nav.no",
		})
		assert.NoError(t, err)

		ds, err := stores.DataProductsStorage.CreateDataset(context.Background(), service.NewDataset{
			DataproductID: dp.ID,
			Name:          "My Dataset",
			Pii:           "none",
			BigQuery: service.NewBigQuery{
				ProjectID: gcpProject,
				Dataset:   "test-dataset",
				Table:     "test-table",
			},
			Metadata: service.BigqueryMetadata{
				Schema: service.BigquerySchema{
					Columns: []*service.BigqueryColumn{
						{
							Name: "id",
							Type: "STRING",
							Mode: "REQUIRED",
						},
					},
				},
				TableType: "TABLE",
			},
		}, nil, user)
		assert.NoError(t, err)

		NewTester(t, server).Post(nil, "/api/bigquery/tables/sync").
			HasStatusCode(http.StatusNoContent)

		expect := &service.BigQuery{
			DatasetID: ds.ID,
			ProjectID: gcpProject,
			Dataset:   "test-dataset",
			Table:     "test-table",
			TableType: "TABLE",
			Schema: []*service.BigqueryColumn{
				{
					Name: "id",
					Type: "STRING",
					Mode: "REQUIRED",
				},
				{
					Name: "name",
					Type: "STRING",
					Mode: "NULLABLE",
				},
				{
					Name: "description",
					Type: "STRING",
					Mode: "NULLABLE",
				},
			},
			PiiTags: strToStrPtr("{}"),
		}

		source, err := stores.BigQueryStorage.GetBigqueryDatasource(context.Background(), ds.ID, false)
		assert.NoError(t, err)
		diff := cmp.Diff(expect, source, cmpopts.IgnoreFields(service.BigQuery{}, "ID", "LastModified", "Created", "Expires", "MissingSince"))
		assert.Empty(t, diff)
	})

	// FIXME: Check sync with pseudo tables
}
