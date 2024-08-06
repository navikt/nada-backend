package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInsightProduct(t *testing.T) {
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

	zlog := zerolog.New(os.Stdout)
	r := TestRouter(zlog)

	{
		store := postgres.NewInsightProductStorage(repo)
		s := core.NewInsightProductService(store)
		h := handlers.NewInsightProductHandler(s)
		e := routes.NewInsightProductEndpoints(zlog, h)
		// This should be configurable per test
		f := routes.NewInsightProductRoutes(e, injectUser(&service.User{
			Email: "bob.the.builder@example.com",
			GoogleGroups: []service.Group{
				{
					Name:  "nada",
					Email: "nada@nav.no",
				},
			},
		}))
		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	var ip *service.InsightProduct

	t.Run("Create Insight Product", func(t *testing.T) {
		insightProduct := &service.NewInsightProduct{
			Name:        "Bob insights",
			Description: strToStrPtr("This is my new insight product"),
			Type:        "Metabase",
			Link:        "https://example.com/something",
			Group:       "nada@nav.no",
		}

		expect := &service.InsightProduct{
			ID:               uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			Name:             "Bob insights",
			Creator:          "bob.the.builder@example.com",
			Description:      "This is my new insight product",
			Type:             "Metabase",
			Link:             "https://example.com/something",
			Keywords:         []string{},
			Group:            "nada@nav.no",
			TeamkatalogenURL: nil,
			TeamID:           nil,
			Created:          time.Time{},
			LastModified:     nil,
			TeamName:         nil,
			ProductAreaName:  "",
		}

		got := &service.InsightProduct{}

		NewTester(t, server).
			Post(insightProduct, "/api/insightProducts/new").
			HasStatusCode(http.StatusOK).
			Expect(expect, got, cmpopts.IgnoreFields(service.InsightProduct{}, "ID", "Created", "LastModified"))

		ip = got
	})

	t.Run("Get Insight Product", func(t *testing.T) {
		NewTester(t, server).Get("/api/insightProducts/"+ip.ID.String()).
			HasStatusCode(http.StatusOK).
			Expect(ip, &service.InsightProduct{})
	})

	t.Run("Update Insight Product", func(t *testing.T) {
		insightProduct := &service.UpdateInsightProductDto{
			Name:        "Bob insights - updated",
			Description: ip.Description,
			TypeArg:     ip.Type,
			Link:        ip.Link,
			Keywords:    []string{"tag1", "tag2"},
			Group:       "nada@nav.no",
		}

		got := &service.InsightProduct{}

		NewTester(t, server).Put(insightProduct, "/api/insightProducts/"+ip.ID.String()).
			HasStatusCode(http.StatusOK).
			Value(got)

		assert.Equal(t, insightProduct.Name, got.Name)
	})

	t.Run("Delete Insight Product", func(t *testing.T) {
		NewTester(t, server).Delete("/api/insightProducts/" + ip.ID.String()).
			HasStatusCode(http.StatusOK)
	})

	t.Run("Get Insight Product - Not Found", func(t *testing.T) {
		NewTester(t, server).Get("/api/insightProducts/" + ip.ID.String()).
			HasStatusCode(http.StatusNotFound)
	})
}
