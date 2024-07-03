package integration

import (
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

	{
		stores := storage.NewStores(repo, config.Config{}, log)
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
		expect := &service.ProductAreasDto{ProductAreas: []*service.ProductArea{}}

		NewTester(t, server).Get("/api/productareas").
			HasStatusCode(http.StatusOK).
			Expect(expect, got)
	})

}
