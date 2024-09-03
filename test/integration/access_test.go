// nolint
package integration

import (
	"context"
	"fmt"
	http2 "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/sa"
	serviceAccountEmulator "github.com/navikt/nada-backend/pkg/sa/emulator"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api/static"
	"github.com/navikt/nada-backend/pkg/syncers/metabase_mapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestAccess(t *testing.T) {
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
	datasetOwnerRouter := TestRouter(zlog)
	accessRequesterRouter := TestRouter(zlog)

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

	userService := core.NewUserService(
		stores.AccessStorage,
		stores.TokenStorage,
		stores.StoryStorage,
		stores.DataProductsStorage,
		stores.InsightProductStorage,
		stores.NaisConsoleStorage,
		zlog,
	)

	StorageCreateProductAreasAndTeams(t, stores.ProductAreaStorage)
	fuel, err := dataproductService.CreateDataproduct(ctx, UserOne, NewDataProductBiofuelProduction(GroupEmailNada, TeamSeagrassID))
	assert.NoError(t, err)

	fuelData, err := dataproductService.CreateDataset(ctx, UserOne, NewDatasetBiofuelConsumptionRates(fuel.ID))
	assert.NoError(t, err)

	{
		h := handlers.NewUserHandler(userService)
		e := routes.NewUserEndpoints(zlog, h)
		fAccessRequesterRoutes := routes.NewUserRoutes(e, injectUser(UserTwo))
		fAccessRequesterRoutes(accessRequesterRouter)
	}

	{
		h := handlers.NewDataProductsHandler(dataproductService)
		e := routes.NewDataProductsEndpoints(zlog, h)
		fDatasetOwnerRoutes := routes.NewDataProductsRoutes(e, injectUser(UserOne))

		fDatasetOwnerRoutes(datasetOwnerRouter)
	}

	{
		h := handlers.NewMetabaseHandler(mbService, mapper.Queue)
		e := routes.NewMetabaseEndpoints(zlog, h)
		fDatasetOwnerRoutes := routes.NewMetabaseRoutes(e, injectUser(UserOne))

		fDatasetOwnerRoutes(datasetOwnerRouter)
	}

	{
		slack := static.NewSlackAPI(log)
		s := core.NewAccessService(
			"https://data.nav.no",
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
		fDatasetOwnerRoutes := routes.NewAccessRoutes(e, injectUser(UserOne))
		fAccessRequesterRoutes := routes.NewAccessRoutes(e, injectUser(UserTwo))

		fDatasetOwnerRoutes(datasetOwnerRouter)
		fAccessRequesterRoutes(accessRequesterRouter)
	}

	datasetOwnerServer := httptest.NewServer(datasetOwnerRouter)
	defer datasetOwnerServer.Close()

	accessRequesterServer := httptest.NewServer(accessRequesterRouter)
	defer accessRequesterServer.Close()

	t.Run("Create dataset access request", func(t *testing.T) {
		NewTester(t, accessRequesterServer).
			Post(service.NewAccessRequestDTO{
				DatasetID:   fuelData.ID,
				Expires:     nil,
				Subject:     strToStrPtr(UserTwo.Email),
				SubjectType: strToStrPtr(service.SubjectTypeUser),
			}, "/api/accessRequests/new").
			HasStatusCode(http2.StatusNoContent)

		expect := &service.AccessRequestsWrapper{
			AccessRequests: []*service.AccessRequest{
				{
					DatasetID:   fuelData.ID,
					Subject:     UserTwoEmail,
					SubjectType: service.SubjectTypeUser,
					Owner:       UserTwoEmail,
					Status:      service.AccessRequestStatusPending,
				},
			},
		}
		got := &service.AccessRequestsWrapper{}

		NewTester(t, datasetOwnerServer).Get("/api/accessRequests", "datasetId", fuelData.ID.String()).
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 1)
		diff := cmp.Diff(expect.AccessRequests[0], got.AccessRequests[0], cmpopts.IgnoreFields(service.AccessRequest{}, "ID", "Created"))
		assert.Empty(t, diff)
	})

	existingAR := &service.AccessRequest{}
	t.Run("Approve dataset access request", func(t *testing.T) {
		existingARs := &service.AccessRequestsWrapper{}
		NewTester(t, datasetOwnerServer).Get("/api/accessRequests", "datasetId", fuelData.ID.String()).
			HasStatusCode(http2.StatusOK).
			Value(existingARs)

		existingAR = existingARs.AccessRequests[0]

		NewTester(t, datasetOwnerServer).Post(nil, fmt.Sprintf("/api/accessRequests/process/%v", existingAR.ID), "action", "approve").
			HasStatusCode(http2.StatusNoContent)

		expect := &service.Dataset{
			Access: []*service.Access{
				{
					DatasetID:       existingAR.DatasetID,
					Subject:         "user:" + UserTwo.Email,
					Owner:           UserTwo.Email,
					Granter:         UserOne.Email,
					AccessRequestID: &existingAR.ID,
				},
			},
		}

		got := &service.Dataset{}
		NewTester(t, datasetOwnerServer).Get(fmt.Sprintf("/api/datasets/%v", existingAR.DatasetID)).
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.Access, 1)
		diff := cmp.Diff(expect.Access[0], got.Access[0], cmpopts.IgnoreFields(service.Access{}, "ID", "Created"))
		assert.Empty(t, diff)
	})

	t.Run("Delete dataset access request", func(t *testing.T) {
		got := &service.UserInfo{}
		NewTester(t, accessRequesterServer).Get("/api/userData").
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 1)

		NewTester(t, accessRequesterServer).Delete(fmt.Sprintf("/api/accessRequests/%v", existingAR.ID)).
			HasStatusCode(http2.StatusNoContent)

		NewTester(t, accessRequesterServer).Get("/api/userData").
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 0)
	})

	t.Run("Deny dataset access request", func(t *testing.T) {
		denyReason := "you must provide a purpose for access to this dataset"
		NewTester(t, accessRequesterServer).
			Post(service.NewAccessRequestDTO{
				DatasetID:   fuelData.ID,
				Expires:     nil,
				Subject:     strToStrPtr(UserTwo.Email),
				SubjectType: strToStrPtr(service.SubjectTypeUser),
			}, "/api/accessRequests/new").
			HasStatusCode(http2.StatusNoContent)

		existingARs := &service.AccessRequestsWrapper{}
		NewTester(t, datasetOwnerServer).Get("/api/accessRequests", "datasetId", fuelData.ID.String()).
			HasStatusCode(http2.StatusOK).
			Value(existingARs)

		ar := existingARs.AccessRequests[0]

		NewTester(t, datasetOwnerServer).Post(nil, fmt.Sprintf("/api/accessRequests/process/%v", ar.ID), "action", "deny", "reason", url.QueryEscape(denyReason)).
			HasStatusCode(http2.StatusNoContent)

		expect := &service.UserInfo{
			AccessRequests: []service.AccessRequest{
				{
					ID:          ar.ID,
					DatasetID:   ar.DatasetID,
					Granter:     strToStrPtr(UserOne.Email),
					Owner:       UserTwo.Email,
					Subject:     UserTwo.Email,
					SubjectType: service.SubjectTypeUser,
					Status:      service.AccessRequestStatusDenied,
					Reason:      &denyReason,
				},
			},
		}

		got := &service.UserInfo{}
		NewTester(t, accessRequesterServer).Get("/api/userData").
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 1)
		diff := cmp.Diff(expect.AccessRequests[0], got.AccessRequests[0], cmpopts.IgnoreFields(service.AccessRequest{}, "Created", "Closed"))
		assert.Empty(t, diff)

		NewTester(t, accessRequesterServer).Delete(fmt.Sprintf("/api/accessRequests/%v", ar.ID)).
			HasStatusCode(http2.StatusNoContent)

		NewTester(t, accessRequesterServer).Get("/api/userData").
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 0)
	})

	t.Run("Grant dataset access request for service account", func(t *testing.T) {
		const serviceaccountName = "my-sa@project-id.iam.gserviceaccount.com"
		NewTester(t, accessRequesterServer).
			Post(service.NewAccessRequestDTO{
				DatasetID:   fuelData.ID,
				Expires:     nil,
				Subject:     strToStrPtr(serviceaccountName),
				SubjectType: strToStrPtr(service.SubjectTypeServiceAccount),
				Owner:       strToStrPtr(GroupEmailAllUsers),
			}, "/api/accessRequests/new").
			HasStatusCode(http2.StatusNoContent)

		existingARs := &service.AccessRequestsWrapper{}
		NewTester(t, datasetOwnerServer).Get("/api/accessRequests", "datasetId", fuelData.ID.String()).
			HasStatusCode(http2.StatusOK).
			Value(existingARs)

		ar := existingARs.AccessRequests[0]

		NewTester(t, datasetOwnerServer).Post(nil, fmt.Sprintf("/api/accessRequests/process/%v", ar.ID), "action", "approve").
			HasStatusCode(http2.StatusNoContent)

		expect := &service.UserInfo{
			AccessRequests: []service.AccessRequest{
				{
					ID:          ar.ID,
					DatasetID:   ar.DatasetID,
					Granter:     strToStrPtr(UserOne.Email),
					Owner:       GroupEmailAllUsers,
					Subject:     serviceaccountName,
					SubjectType: service.SubjectTypeServiceAccount,
					Status:      service.AccessRequestStatusApproved,
				},
			},
			Accessable: service.AccessibleDatasets{
				ServiceAccountGranted: []*service.AccessibleDataset{
					{
						Subject: strToStrPtr("serviceAccount:" + serviceaccountName),
						Dataset: service.Dataset{
							ID:            fuelData.ID,
							DataproductID: fuel.ID,
						},
					},
				},
			},
		}

		got := &service.UserInfo{}
		NewTester(t, accessRequesterServer).Get("/api/userData").
			HasStatusCode(http2.StatusOK).
			Value(got)

		require.Len(t, got.AccessRequests, 1)
		diff := cmp.Diff(expect.AccessRequests[0], got.AccessRequests[0], cmpopts.IgnoreFields(service.AccessRequest{}, "Created", "Closed"))
		assert.Empty(t, diff)

		require.Len(t, got.Accessable.ServiceAccountGranted, 1)
		assert.Equal(t, *expect.Accessable.ServiceAccountGranted[0].Subject, *got.Accessable.ServiceAccountGranted[0].Subject)
		assert.Equal(t, expect.Accessable.ServiceAccountGranted[0].DataproductID, got.Accessable.ServiceAccountGranted[0].DataproductID)
		assert.Equal(t, expect.Accessable.ServiceAccountGranted[0].ID, got.Accessable.ServiceAccountGranted[0].ID)
	})
}
