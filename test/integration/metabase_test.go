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

const fakeMetabaseSA = `{
  "type": "service_account",
  "project_id": "test",
  "private_key_id": "very-long-key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCQ+2XPvBgrF0rC\nSNsZxYgyrTxtr47z32yKO38Hi4ViRbDZVMr2isz2gnEW3MgwzCWEDuZqFkpigAUS\n9JPTHFXbNgHv28mAcoNEUBlabmzGJG+MUL3ZbiwjOVRhAyZXexFb9XTNk/0a30tF\nPw2HHtJYKda/C0ELitOBjfz74yhVXN8U16gEU23eRAmUb82yb2v2gy9pe8CjB5AZ\n8iOEtCACte5uwpoSSpb9wUZNcI4uwlX9jUDrLP7eFeAhaYz7xSMuHHGYM9P4rdcN\n77OGB8oA3k1A+sI6vnZaU/++RvbhHkK7xVQdlrEWHd9Mkw1SBTN4TIkxAcdn1fPv\ngujC4pbNAgMBAAECggEAEeWWqdAUQ616YwVPVp0VtPvBi+CyCtS3t1Ck26+TZZAo\nxos7CrgTmRJ9F09lvvBUaTrVop7hy8r11WMvpE3JI2hFUPCWrS51mccxegJLlyIE\nSxPke6Sn+ikni1oyL3ZXrDxekMoF4n1R81hXOSjK2ms/wRGIk/4tIb7/TbC6196R\ncWtT9unK1tWpaIzBaC0MH0/hnKKC9Vq9g+ezdnFo8eesdOiBoO8R5aAOofcA00Fm\n47RW0BoQNwIeyv6YcD2jg3cg8/FzODEktOW3/WP/Svr6FFPeQKKp3wjRN/ovU4kS\n/tDsHrYa5cSrpuI1dkzqHbu3AbQBV/NNd9pzJwAfwQKBgQDDn/8D1BhllPPTqOSD\n/2Z51MpZ+1OLkp11zFLUCnxsIop1wQHI58CTdn6+6Z4z3hEpJVkcxumkH0LMFHnH\n0ZZt8PaaqMQuIhxJZC4t8a/PuAGDK2G7kvusoSq92AVZjwlrIJ9Ij/RFE2Qoyo1o\n+9tLGUI9TMBAqcsdFReUxIAZOQKBgQC9ui999G5d5S0Dyg8+qGNf/QPM/yVfSA3d\nViCw77c/UiHG4mDa/leMBMLunKp00x8aE4dpbZYP138Jwzm1YS1i/ifqNEfdu5it\nLeNKHe6lEWnjW7j31Hn9oQbSG5BNtpHyII/1qs2YwmANWmh8o42SuNnrxoq3qlbd\n13KC2B5ONQKBgGvChdKREgNbAtlUTtTbapKwAeuBQ2s+D1jlfbbqM9HJUSY+dII8\nD1vryTPXMtt1d1SIC0eL1wYeZkhO+yp0LH5RXzagwrh698QB2GJcoTE2NjcQPZz7\nAYH9obLD/WZxIYoOhU+OZMtsPB8wPKdZHVqIBnIIBltYbNePV9cOS1YZAoGBALV3\nDhOfpaxDFbIJIlmgvyPBIVCCPWGLzk8EINJ7BT8YNFxAi7kKCfxPVY7Z46NHhvju\n8tZgzWWrjMNuqZSVNM75Hn5AsPgghOAnAr0SMf5J0Ih4Y0sPO/rdeGOfn37k/2Sh\nxm+HhYv1Zd9/uG52FGPgT/bV+DnBP8KBXfJN+XZ9AoGBAIPl1Ue+ZF3aUGPEwaZk\nMPeLEvGpVfxgBs8iJszJ+PXerNlZW1Bf+Xz3M0nlQF5vV7IdvcpMdGLBEhqxbijM\n7i0RnotQpqGObSoezJivZTmy8Yn6apH5HBAwlmrSYsqYJeWfWNdMhaYiCSOsq1eT\nlz7CEBohLrUvBJZe2b0MRFIJ\n-----END PRIVATE KEY-----\n",
  "client_email": "nada-metabase@test.iam.gserviceaccount.com",
  "client_id": "very-long-client-id",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/nada-metabase%40test.iam.gserviceaccount.com"
}
`

func TestMetabase(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(20*time.Minute))
	defer cancel()

	log := zerolog.New(zerolog.NewConsoleWriter())

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
				Members: []string{fmt.Sprintf("user:%s", GroupNada)},
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
		GroupAllUsers,
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
		GroupAllUsers,
	)

	StorageCreateProductAreasAndTeams(t, stores.ProductAreaStorage)
	fuel, err := dataproductService.CreateDataproduct(ctx, TestUser, NewDataProductBiofuelProduction())
	assert.NoError(t, err)

	fuelData, err := dataproductService.CreateDataset(ctx, TestUser, NewDatasetBiofuelConsumptionRates(fuel.ID))
	assert.NoError(t, err)

	{
		h := handlers.NewMetabaseHandler(mbService, mapper.Queue)
		e := routes.NewMetabaseEndpoints(zlog, h)
		f := routes.NewMetabaseRoutes(e, injectUser(TestUser))

		f(r)
	}

	{
		slack := static.NewSlackAPI(log)
		s := core.NewAccessService(
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
		f := routes.NewAccessRoutes(e, injectUser(TestUser))

		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("Adding a restricted dataset to metabase", func(t *testing.T) {
		NewTester(t, server).
			Post(service.GrantAccessData{
				DatasetID:   fuelData.ID,
				Expires:     nil,
				Subject:     strToStrPtr(TestUser.Email),
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
		go collectionSyncer.Run(ctx)
		time.Sleep(5 * time.Second)

		collections, err := mbapi.GetCollections(ctx)
		require.NoError(t, err)
		assert.True(t, ContainsCollectionWithName(collections, "My new collection name 🔐"))
	})
}
