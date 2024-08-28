// nolint
package integration

// import (
// 	"net/http/httptest"
// 	"os"
// 	"testing"
//
// 	"github.com/navikt/nada-backend/pkg/bq"
// 	"github.com/navikt/nada-backend/pkg/bq/emulator"
// 	"github.com/navikt/nada-backend/pkg/database"
// 	"github.com/navikt/nada-backend/pkg/service"
// 	"github.com/navikt/nada-backend/pkg/service/core"
// 	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
// 	"github.com/navikt/nada-backend/pkg/service/core/api/http"
// 	"github.com/navikt/nada-backend/pkg/service/core/handlers"
// 	"github.com/navikt/nada-backend/pkg/service/core/routes"
// 	"github.com/navikt/nada-backend/pkg/service/core/storage"
// 	"github.com/rs/zerolog"
// )
//
// const fakeMetabaseSA = `{
//   "type": "service_account",
//   "project_id": "test",
//   "private_key_id": "very-long-key-id",
//   "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCQ+2XPvBgrF0rC\nSNsZxYgyrTxtr47z32yKO38Hi4ViRbDZVMr2isz2gnEW3MgwzCWEDuZqFkpigAUS\n9JPTHFXbNgHv28mAcoNEUBlabmzGJG+MUL3ZbiwjOVRhAyZXexFb9XTNk/0a30tF\nPw2HHtJYKda/C0ELitOBjfz74yhVXN8U16gEU23eRAmUb82yb2v2gy9pe8CjB5AZ\n8iOEtCACte5uwpoSSpb9wUZNcI4uwlX9jUDrLP7eFeAhaYz7xSMuHHGYM9P4rdcN\n77OGB8oA3k1A+sI6vnZaU/++RvbhHkK7xVQdlrEWHd9Mkw1SBTN4TIkxAcdn1fPv\ngujC4pbNAgMBAAECggEAEeWWqdAUQ616YwVPVp0VtPvBi+CyCtS3t1Ck26+TZZAo\nxos7CrgTmRJ9F09lvvBUaTrVop7hy8r11WMvpE3JI2hFUPCWrS51mccxegJLlyIE\nSxPke6Sn+ikni1oyL3ZXrDxekMoF4n1R81hXOSjK2ms/wRGIk/4tIb7/TbC6196R\ncWtT9unK1tWpaIzBaC0MH0/hnKKC9Vq9g+ezdnFo8eesdOiBoO8R5aAOofcA00Fm\n47RW0BoQNwIeyv6YcD2jg3cg8/FzODEktOW3/WP/Svr6FFPeQKKp3wjRN/ovU4kS\n/tDsHrYa5cSrpuI1dkzqHbu3AbQBV/NNd9pzJwAfwQKBgQDDn/8D1BhllPPTqOSD\n/2Z51MpZ+1OLkp11zFLUCnxsIop1wQHI58CTdn6+6Z4z3hEpJVkcxumkH0LMFHnH\n0ZZt8PaaqMQuIhxJZC4t8a/PuAGDK2G7kvusoSq92AVZjwlrIJ9Ij/RFE2Qoyo1o\n+9tLGUI9TMBAqcsdFReUxIAZOQKBgQC9ui999G5d5S0Dyg8+qGNf/QPM/yVfSA3d\nViCw77c/UiHG4mDa/leMBMLunKp00x8aE4dpbZYP138Jwzm1YS1i/ifqNEfdu5it\nLeNKHe6lEWnjW7j31Hn9oQbSG5BNtpHyII/1qs2YwmANWmh8o42SuNnrxoq3qlbd\n13KC2B5ONQKBgGvChdKREgNbAtlUTtTbapKwAeuBQ2s+D1jlfbbqM9HJUSY+dII8\nD1vryTPXMtt1d1SIC0eL1wYeZkhO+yp0LH5RXzagwrh698QB2GJcoTE2NjcQPZz7\nAYH9obLD/WZxIYoOhU+OZMtsPB8wPKdZHVqIBnIIBltYbNePV9cOS1YZAoGBALV3\nDhOfpaxDFbIJIlmgvyPBIVCCPWGLzk8EINJ7BT8YNFxAi7kKCfxPVY7Z46NHhvju\n8tZgzWWrjMNuqZSVNM75Hn5AsPgghOAnAr0SMf5J0Ih4Y0sPO/rdeGOfn37k/2Sh\nxm+HhYv1Zd9/uG52FGPgT/bV+DnBP8KBXfJN+XZ9AoGBAIPl1Ue+ZF3aUGPEwaZk\nMPeLEvGpVfxgBs8iJszJ+PXerNlZW1Bf+Xz3M0nlQF5vV7IdvcpMdGLBEhqxbijM\n7i0RnotQpqGObSoezJivZTmy8Yn6apH5HBAwlmrSYsqYJeWfWNdMhaYiCSOsq1eT\nlz7CEBohLrUvBJZe2b0MRFIJ\n-----END PRIVATE KEY-----\n",
//   "client_email": "nada-metabase@test.iam.gserviceaccount.com",
//   "client_id": "very-long-client-id",
//   "auth_uri": "https://accounts.google.com/o/oauth2/auth",
//   "token_uri": "https://oauth2.googleapis.com/token",
//   "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
//   "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/nada-metabase%40test.iam.gserviceaccount.com"
// }
// `
//
// func TestMetabase(t *testing.T) {
// 	log := zerolog.New(os.Stdout)
// 	c := NewContainers(t, log)
// 	defer c.Cleanup()
//
// 	pgCfg := c.RunPostgres(NewPostgresConfig())
//
// 	_, err := database.New(
// 		pgCfg.ConnectionURL(),
// 		10,
// 		10,
// 	)
// 	assert.NoError(t, err)
//
// 	mbCfg := c.RunMetabase(NewMetabaseConfig())
//
// 	gcpProject := "test-project"
// 	gcpLocation := "europe-north1"
// 	datasets := []*emulator.Dataset{
// 		{
// 			DatasetID: "test-dataset",
// 			TableID:   "test-table",
// 			Columns: []*types.Column{
// 				emulator.ColumnRequired("id"),
// 				emulator.ColumnNullable("name"),
// 				emulator.ColumnNullable("description"),
// 			},
// 		},
// 		{
// 			DatasetID: "pseudo-test-dataset",
// 		},
// 	}
//
// 	em := emulator.New(log)
//
// 	em.WithProject(gcpProject, datasets...)
// 	em.TestServer()
// 	bqClient := bq.NewClient(em.Endpoint(), false)
//
// 	stores := storage.NewStores(repo, config.Config{}, log)
//
// 	zlog := zerolog.New(os.Stdout)
// 	r := TestRouter(zlog)
//
// 	{
// 		saapi := gcp.NewServiceAccountAPI()
// 		bqapi := gcp.NewBigQueryAPI(gcpProject, gcpLocation, "pseudo-test-dataset", bqClient)
// 		mbapi := http.NewMetabaseHTTP()
//
// 		s := core.NewMetabaseService(
// 			gcpProject,
// 			fakeMetabaseSA,
// 			"nada-metabase@test.iam.gserviceaccount.com",
// 			"all-users@nav.no",
// 			mbapi,
// 			bqapi,
// 			saapi,
// 			stores.ThirdPartyMappingStorage,
// 			stores.MetaBaseStorage,
// 			stores.BigQueryStorage,
// 			stores.DataProductsStorage,
// 			stores.AccessStorage,
// 		)
// 		h := handlers.NewMetabaseHandler(s)
// 		e := routes.NewMetabaseEndpoints(zlog, h)
// 		f := routes.NewMetabaseRoutes(e, injectUser(
// 			&service.User{
// 				Name:  "Mr. Bob",
// 				Email: "bob@nav.no",
// 				GoogleGroups: []service.Group{
// 					{
// 						Name:  "nada",
// 						Email: "nada@nav.no",
// 					},
// 				},
// 			},
// 		))
//
// 		f(r)
// 	}
//
// 	server := httptest.NewServer(r)
// 	defer server.Close()
// }
