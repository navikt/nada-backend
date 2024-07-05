//go:build integration_test

package integration

import (
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/cs"
	"github.com/navikt/nada-backend/pkg/cs/emulator"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	defaultHtml = "<html><h1>Story</h1></html>"
)

func TestStory(t *testing.T) {
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

	pa1 := uuid.MustParse("00000000-1111-0000-0000-000000000000")
	pa2 := uuid.MustParse("00000000-2222-0000-0000-000000000000")
	team1 := uuid.MustParse("00000000-0000-1111-0000-000000000000")
	team2 := uuid.MustParse("00000000-0000-2222-0000-000000000000")
	team3 := uuid.MustParse("00000000-0000-3333-0000-000000000000")

	pas := []*tk.ProductArea{
		{
			ID:   pa1,
			Name: "Product area 1",
		},
		{
			ID:   pa2,
			Name: "Product area 2",
		},
	}

	teams := []*tk.Team{
		{
			ID:            team1,
			Name:          "Team1",
			Description:   "This is the first team",
			ProductAreaID: pa1,
		},
		{
			ID:            team2,
			Name:          "Team 2",
			Description:   "This is the second team",
			ProductAreaID: pa2,
		},
		{
			ID:            team3,
			Name:          "Team 3",
			Description:   "This is the third team",
			NaisTeams:     []string{"team3"},
			ProductAreaID: pa2,
		},
	}

	staticFetcher := tk.NewStatic("http://example.com", pas, teams)

	router := TestRouter(log)
	e := emulator.New(t, nil)
	e.CreateBucket("nada-backend-stories")
	defer e.Cleanup()

	user := &service.User{
		Name:        "Bob the Builder",
		Email:       "bob.the.builder@nav.no",
		AzureGroups: nil,
		GoogleGroups: []service.Group{
			{
				Email: "nada@nav.no",
				Name:  "nada",
			},
		},
		AllGoogleGroups: nil,
	}

	{
		teamKatalogenAPI := httpapi.NewTeamKatalogenAPI(staticFetcher)
		cs := cs.NewFromClient("nada-backend-stories", e.Client())
		storyAPI := gcp.NewStoryAPI(cs, log)
		tokenService := core.NewTokenService(postgres.NewTokenStorage(repo))
		storyService := core.NewStoryService(postgres.NewStoryStorage(repo), teamKatalogenAPI, storyAPI)
		h := handlers.NewStoryHandler(storyService, tokenService, log)
		e := routes.NewStoryEndpoints(log, h)
		f := routes.NewStoryRoutes(e, injectUser(user), h.NadaTokenMiddleware)
		f(router)
	}

	server := httptest.NewServer(router)

	story := &service.Story{}

	t.Run("Create story with oauth", func(t *testing.T) {
		newStory := &service.NewStory{
			Name:          "My new story",
			Description:   strToStrPtr("This is my story, and it is pretty bad"),
			Keywords:      []string{"story", "bad"},
			ProductAreaID: &pa1,
			TeamID:        &team1,
			Group:         "nada@nav.no",
		}

		expect := &service.Story{
			Name:             "My new story",
			Creator:          "bob.the.builder@nav.no",
			Description:      "This is my story, and it is pretty bad",
			Keywords:         []string{"story", "bad"},
			TeamkatalogenURL: nil,
			TeamID:           &team1,
			Group:            "nada@nav.no",
			// FIXME: can't set these from CreateStory, should they be?
			// TeamName:         strToStrPtr("Team1"),
			// ProductAreaName:  "Product area 1",
		}

		files := map[string]string{
			"index.html": defaultHtml,
		}
		objects := map[string]string{
			"nada-backend-new-story": string(Marshal(t, newStory)),
		}

		req := CreateMultipartFormRequest(t, http.MethodPost, server.URL+"/api/stories/new", files, objects)

		NewTester(t, server).Send(req).
			HasStatusCode(http.StatusOK).
			Expect(expect, story, cmpopts.IgnoreFields(service.Story{}, "ID", "Created", "LastModified"))
	})
}
