// nolint
package integration

import (
	"context"
	"fmt"
	http2 "net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/sa"
	serviceAccountEmulator "github.com/navikt/nada-backend/pkg/sa/emulator"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api/static"
	"github.com/navikt/nada-backend/pkg/syncers/metabase_collections"
	"github.com/navikt/nada-backend/pkg/syncers/metabase_mapper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/cloudresourcemanager/v1"

	"github.com/navikt/nada-backend/pkg/bq"
	bigQueryEmulator "github.com/navikt/nada-backend/pkg/bq/emulator"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	"github.com/navikt/nada-backend/pkg/service/core/api/http"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/rs/zerolog"
)

func TestMetabase(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(20*time.Minute))
	defer cancel()

	log := zerolog.New(zerolog.NewConsoleWriter())
	log.Level(zerolog.DebugLevel)

	c := NewContainers(t, log)
	defer c.Cleanup()

	pgCfg := c.RunPostgres(NewPostgresConfig())

	repo, err := database.New(
		pgCfg.ConnectionURL(),
		10,
		10,
	)
	assert.NoError(t, err)

	mbCfg := c.RunMetabase(NewMetabaseConfig())

	fuelBqSchema := NewDatasetBiofuelConsumptionRatesSchema()

	bqe := bigQueryEmulator.New(log)
	bqe.WithProject(Project, fuelBqSchema...)
	bqe.EnableMock(false, log, bigQueryEmulator.NewPolicyMock(log).Mocks()...)

	bqHTTPPort := strconv.Itoa(GetFreePort(t))
	bqHTTPAddr := fmt.Sprintf("127.0.0.1:%s", bqHTTPPort)
	if len(os.Getenv("CI")) > 0 {
		bqHTTPAddr = fmt.Sprintf("0.0.0.0:%s", bqHTTPPort)
	}
	bqGRPCAddr := fmt.Sprintf("127.0.0.1:%s", strconv.Itoa(GetFreePort(t)))
	go func() {
		_ = bqe.Serve(ctx, bqHTTPAddr, bqGRPCAddr)
	}()
	bqClient := bq.NewClient("http://"+bqHTTPAddr, false, log)

	saEmulator := serviceAccountEmulator.New(log)
	saEmulator.SetPolicy(Project, &cloudresourcemanager.Policy{
		Bindings: []*cloudresourcemanager.Binding{
			{
				Role:    "roles/owner",
				Members: []string{fmt.Sprintf("user:%s", GroupEmailNada)},
			},
		},
	})
	saURL := saEmulator.Run()
	saClient := sa.NewClient(saURL, true)

	stores := storage.NewStores(repo, config.Config{}, log)

	zlog := zerolog.New(os.Stdout)
	r := TestRouter(zlog)

	bigQueryContainerHostPort := "http://host.docker.internal:" + bqHTTPPort
	if len(os.Getenv("CI")) > 0 {
		bigQueryContainerHostPort = "http://172.17.0.1:" + bqHTTPPort
	}

	saapi := gcp.NewServiceAccountAPI(saClient)
	bqapi := gcp.NewBigQueryAPI(Project, Location, PseudoDataSet, bqClient)
	// FIXME: should we just add /api to the connectionurl returned
	mbapi := http.NewMetabaseHTTP(
		mbCfg.ConnectionURL()+"/api",
		mbCfg.Email,
		mbCfg.Password,
		// We want metabase to connect with the big query emulator
		// running on the host
		bigQueryContainerHostPort,
		true,
		false,
		log,
	)

	mbService := core.NewMetabaseService(
		Project,
		fakeMetabaseSA,
		"nada-metabase@test.iam.gserviceaccount.com",
		GroupEmailAllUsers,
		mbapi,
		bqapi,
		saapi,
		stores.ThirdPartyMappingStorage,
		stores.MetaBaseStorage,
		stores.BigQueryStorage,
		stores.DataProductsStorage,
		stores.AccessStorage,
		zlog,
	)

	mapper := metabase_mapper.New(mbService, stores.ThirdPartyMappingStorage, 60, 60, log)
	assert.NoError(t, err)
	go mapper.Run(ctx)

	err = stores.NaisConsoleStorage.UpdateAllTeamProjects(ctx, map[string]string{
		NaisTeamNada: Project,
	})
	assert.NoError(t, err)

	dataproductService := core.NewDataProductsService(
		stores.DataProductsStorage,
		stores.BigQueryStorage,
		bqapi,
		stores.NaisConsoleStorage,
		GroupEmailAllUsers,
	)

	StorageCreateProductAreasAndTeams(t, stores.ProductAreaStorage)
	fuel, err := dataproductService.CreateDataproduct(ctx, UserOne, NewDataProductBiofuelProduction(GroupEmailNada, TeamSeagrassID))
	assert.NoError(t, err)

	fuelData, err := dataproductService.CreateDataset(ctx, UserOne, NewDatasetBiofuelConsumptionRates(fuel.ID))
	assert.NoError(t, err)

	{
		h := handlers.NewMetabaseHandler(mbService, mapper.Queue)
		e := routes.NewMetabaseEndpoints(zlog, h)
		f := routes.NewMetabaseRoutes(e, injectUser(UserOne))

		f(r)
	}

	{
		slack := static.NewSlackAPI(log)
		s := core.NewAccessService(
			"",
			slack,
			stores.PollyStorage,
			stores.AccessStorage,
			stores.DataProductsStorage,
			stores.BigQueryStorage,
			stores.JoinableViewsStorage,
			bqapi,
		)
		h := handlers.NewAccessHandler(s, mbService, Project)
		e := routes.NewAccessEndpoints(zlog, h)
		f := routes.NewAccessRoutes(e, injectUser(UserOne))

		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("Adding a restricted dataset to metabase", func(t *testing.T) {
		NewTester(t, server).
			Post(service.GrantAccessData{
				DatasetID:   fuelData.ID,
				Expires:     nil,
				Subject:     strToStrPtr(UserOne.Email),
				SubjectType: strToStrPtr(service.SubjectTypeUser),
			}, "/api/accesses/grant").
			HasStatusCode(http2.StatusNoContent)

		NewTester(t, server).
			Post(service.DatasetMap{Services: []string{service.MappingServiceMetabase}}, fmt.Sprintf("/api/datasets/%s/map", fuelData.ID)).
			HasStatusCode(http2.StatusAccepted)

		// Need to give time for the mapping to be processed, not great
		// perhaps we can bypass the queue in the test
		time.Sleep(20 * time.Second)

		meta, err := stores.MetaBaseStorage.GetMetadata(ctx, fuelData.ID, false)
		require.NoError(t, err)
		require.NotNil(t, meta.SyncCompleted)

		collections, err := mbapi.GetCollections(ctx)
		require.NoError(t, err)
		assert.True(t, ContainsCollectionWithName(collections, "Biofuel Consumption Rates 🔐"))

		permissionGroups, err := mbapi.GetPermissionGroups(ctx)
		require.NoError(t, err)
		assert.True(t, ContainsPermissionGroupWithNamePrefix(permissionGroups, "biofuel-consumption-rates"))

		serviceAccount := saEmulator.GetServiceAccounts()
		assert.Len(t, serviceAccount, 1)
		assert.True(t, ContainsServiceAccount(serviceAccount, "nada-", "@test-project.iam.gserviceaccount.com"))

		serviceAccountKeys := saEmulator.GetServiceAccountKeys()
		assert.Len(t, serviceAccountKeys, 1)

		projectPolicy := saEmulator.GetPolicy(Project)
		assert.Len(t, projectPolicy.Bindings, 2)
		assert.Equal(t, projectPolicy.Bindings[1].Role, "projects/test-project/roles/nada.metabase")
	})

	t.Run("Removing 🔐 is added back", func(t *testing.T) {
		meta, err := stores.MetaBaseStorage.GetMetadata(ctx, fuelData.ID, false)
		require.NoError(t, err)

		err = mbapi.UpdateCollection(ctx, &service.MetabaseCollection{
			ID:          *meta.CollectionID,
			Name:        "My new collection name",
			Description: "My new collection description",
		})
		require.NoError(t, err)

		collectionSyncer := metabase_collections.New(mbapi, stores.MetaBaseStorage, 1, log)
		go collectionSyncer.Run(ctx, 0)
		time.Sleep(5 * time.Second)

		collections, err := mbapi.GetCollections(ctx)
		require.NoError(t, err)
		assert.True(t, ContainsCollectionWithName(collections, "My new collection name 🔐"))
	})
}
