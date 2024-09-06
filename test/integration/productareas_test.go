package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestProductArea(t *testing.T) {
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

	r := TestRouter(log)

	stores := storage.NewStores(repo, config.Config{}, log)

	StorageCreateProductAreasAndTeams(t, stores.ProductAreaStorage)
	fuel := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductBiofuelProduction(GroupEmailNada, TeamSeagrassID))
	feed := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductAquacultureFeed(GroupEmailNada, TeamSeagrassID))
	_ = StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductReefMonitoring(GroupEmailNada, TeamReefID))
	_ = StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductProtectiveBarriers(GroupEmailNada, TeamReefID))

	{
		s := core.NewProductAreaService(stores.ProductAreaStorage, stores.DataProductsStorage, stores.InsightProductStorage, stores.StoryStorage)
		h := handlers.NewProductAreasHandler(s)
		e := routes.NewProductAreaEndpoints(log, h)
		f := routes.NewProductAreaRoutes(e)
		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("Get product areas", func(t *testing.T) {
		got := &service.ProductAreasDto{}
		expect := &service.ProductAreasDto{ProductAreas: []*service.ProductArea{
			{
				TeamkatalogenProductArea: &service.TeamkatalogenProductArea{
					ID:   ProductAreaOceanicID,
					Name: ProductAreaOceanicName,
				},
				Teams: []*service.Team{
					{
						TeamkatalogenTeam: &service.TeamkatalogenTeam{
							ID:            TeamSeagrassID,
							Name:          TeamSeagrassName,
							ProductAreaID: ProductAreaOceanicID,
						},
						DataproductsNumber: 2,
					},
				},
			},
			{
				TeamkatalogenProductArea: &service.TeamkatalogenProductArea{
					ID:   ProductAreaCostalID,
					Name: ProductAreaCostalName,
				},
				Teams: []*service.Team{
					{
						TeamkatalogenTeam: &service.TeamkatalogenTeam{
							ID:            TeamReefID,
							Name:          TeamReefName,
							ProductAreaID: ProductAreaCostalID,
						},
						DataproductsNumber: 2,
					},
				},
			},
		}}

		NewTester(t, server).Get("/api/productareas").
			HasStatusCode(http.StatusOK).
			Expect(expect, got)
	})

	t.Run("Get product area with no id should return 404", func(t *testing.T) {
		NewTester(t, server).Get("/api/productareas/00000000-0000-0000-0000-000000000000").
			HasStatusCode(http.StatusNotFound)
	})

	t.Run("Get product area with assets", func(t *testing.T) {
		got := &service.ProductAreaWithAssets{}
		expect := &service.ProductAreaWithAssets{
			ProductArea: &service.ProductArea{
				TeamkatalogenProductArea: &service.TeamkatalogenProductArea{
					ID:   ProductAreaOceanicID,
					Name: ProductAreaOceanicName,
				},
			},
			Teams: []*service.TeamWithAssets{
				{
					TeamkatalogenTeam: &service.TeamkatalogenTeam{
						ID:            TeamSeagrassID,
						Name:          TeamSeagrassName,
						ProductAreaID: ProductAreaOceanicID,
					},
					Dataproducts: []*service.Dataproduct{
						{
							ID:           feed.ID,
							Name:         feed.Name,
							Created:      feed.Created,
							LastModified: feed.LastModified,
							Description:  feed.Description,
							Slug:         feed.Slug,
							Owner: &service.DataproductOwner{
								Group:         GroupEmailNada,
								TeamID:        feed.Owner.TeamID,
								ProductAreaID: &ProductAreaOceanicID,
							},
							Keywords:        []string{},
							TeamName:        &TeamSeagrassName,
							ProductAreaName: ProductAreaOceanicName,
						},
						{
							ID:           fuel.ID,
							Name:         fuel.Name,
							Created:      fuel.Created,
							LastModified: fuel.LastModified,
							Description:  fuel.Description,
							Slug:         fuel.Slug,
							Owner: &service.DataproductOwner{
								Group:         GroupEmailNada,
								TeamID:        fuel.Owner.TeamID,
								ProductAreaID: &ProductAreaOceanicID,
							},
							Keywords:        []string{},
							TeamName:        &TeamSeagrassName,
							ProductAreaName: ProductAreaOceanicName,
						},
					},
					Stories:         []*service.Story{},
					InsightProducts: []*service.InsightProduct{},
				},
			},
		}

		NewTester(t, server).Get(fmt.Sprintf("/api/productareas/%s", ProductAreaOceanicID.String())).
			HasStatusCode(http.StatusOK).
			Expect(expect, got)
	})
}
