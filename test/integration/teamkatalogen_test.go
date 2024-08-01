package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	"github.com/navikt/nada-backend/pkg/service/core/cache/postgres"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTeamKatalogen(t *testing.T) {
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
	cacher := cache.New(10*time.Second, repo.GetDB(), zlog)

	pas := []*tk.ProductArea{
		{
			ID:   uuid.MustParse("00000000-1111-0000-0000-000000000000"),
			Name: "Product area 1",
		},
		{
			ID:   uuid.MustParse("00000000-2222-0000-0000-000000000001"),
			Name: "Product area 2",
		},
	}

	teams := []*tk.Team{
		{
			ID:            uuid.MustParse("00000000-0000-1111-0000-000000000000"),
			Name:          "Team1",
			Description:   "This is the first team",
			ProductAreaID: uuid.MustParse("00000000-1111-0000-0000-000000000000"),
		},
		{
			ID:            uuid.MustParse("00000000-0000-2222-0000-000000000000"),
			Name:          "Team 2",
			Description:   "This is the second team",
			ProductAreaID: uuid.MustParse("00000000-2222-0000-0000-000000000000"),
		},
		{
			ID:            uuid.MustParse("00000000-0000-3333-0000-000000000000"),
			Name:          "Team 3",
			Description:   "This is the third team",
			NaisTeams:     []string{"team3"},
			ProductAreaID: uuid.MustParse("00000000-2222-0000-0000-000000000000"),
		},
	}

	staticFetcher := tk.NewStatic("http://example.com", pas, teams)

	router := TestRouter(zlog)

	{
		apiTk := httpapi.NewTeamKatalogenAPI(staticFetcher, log)
		cacheTk := postgres.NewTeamKatalogenCache(apiTk, cacher)
		s := core.NewTeamKatalogenService(cacheTk)
		h := handlers.NewTeamKatalogenHandler(s)
		e := routes.NewTeamkatalogenEndpoints(zlog, h)
		f := routes.NewTeamkatalogenRoutes(e)
		f(router)
	}

	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("Search", func(t *testing.T) {
		expect := []service.TeamkatalogenResult{
			{
				TeamID:        "00000000-0000-1111-0000-000000000000",
				Name:          "Team1",
				Description:   "This is the first team",
				ProductAreaID: "00000000-1111-0000-0000-000000000000",
			},
			{
				TeamID:        "00000000-0000-3333-0000-000000000000",
				Name:          "Team 3",
				Description:   "This is the third team",
				ProductAreaID: "00000000-2222-0000-0000-000000000000",
			},
		}

		got := []service.TeamkatalogenResult{}

		// FIXME: we seem to have some problem with query parameters when they are encoded
		// need to figure that out..
		NewTester(t, server).Get("/api/teamkatalogen", "gcpGroups", "Team1", "gcpGroups", "team3").
			HasStatusCode(http.StatusOK).
			Expect(&expect, &got)

		stats := cacher.Stats()
		assert.Equal(t, stats.TotalHits, 0)
		assert.Equal(t, stats.TotalMisses, 1)
	})

	t.Run("Search with cache hit", func(t *testing.T) {
		expect := []service.TeamkatalogenResult{
			{
				TeamID:        "00000000-0000-1111-0000-000000000000",
				Name:          "Team1",
				Description:   "This is the first team",
				ProductAreaID: "00000000-1111-0000-0000-000000000000",
			},
			{
				TeamID:        "00000000-0000-3333-0000-000000000000",
				Name:          "Team 3",
				Description:   "This is the third team",
				ProductAreaID: "00000000-2222-0000-0000-000000000000",
			},
		}

		got := []service.TeamkatalogenResult{}

		NewTester(t, server).Get("/api/teamkatalogen", "gcpGroups", "Team1", "gcpGroups", "team3").
			HasStatusCode(http.StatusOK).
			Expect(&expect, &got)

		stats := cacher.Stats()
		assert.Equal(t, stats.TotalHits, 1)
		assert.Equal(t, stats.TotalMisses, 1)
	})
}
