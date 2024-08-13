package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	nada := uuid.MustParse("00000000-0000-4444-0000-000000000000")

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
		{
			ID:            nada,
			Name:          "nada",
			Description:   "Nada team",
			NaisTeams:     []string{"nada"},
			ProductAreaID: pa1,
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

	naisConsoleStorage := postgres.NewNaisConsoleStorage(repo)
	err = naisConsoleStorage.UpdateAllTeamProjects(context.Background(), map[string]string{
		"nada": "gcp-project-team1",
	})
	assert.NoError(t, err)

	tokenStorage := postgres.NewTokenStorage(repo)

	{
		teamKatalogenAPI := httpapi.NewTeamKatalogenAPI(staticFetcher, log)
		cs := cs.NewFromClient("nada-backend-stories", e.Client())
		storyAPI := gcp.NewStoryAPI(cs, log)
		tokenService := core.NewTokenService(tokenStorage)
		storyService := core.NewStoryService(postgres.NewStoryStorage(repo), teamKatalogenAPI, storyAPI, false)
		h := handlers.NewStoryHandler("@nav.no", storyService, tokenService, log)
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
			TeamID:        &nada,
			Group:         "nada@nav.no",
		}

		expect := &service.Story{
			Name:             "My new story",
			Creator:          "bob.the.builder@nav.no",
			Description:      "This is my story, and it is pretty bad",
			Keywords:         []string{"story", "bad"},
			TeamkatalogenURL: nil,
			TeamID:           &nada,
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

		req := CreateMultipartFormRequest(t, http.MethodPost, server.URL+"/api/stories/new", files, objects, nil)

		NewTester(t, server).Send(req).
			HasStatusCode(http.StatusOK).
			Expect(expect, story, cmpopts.IgnoreFields(service.Story{}, "ID", "Created", "LastModified"))
	})

	t.Run("Update story with oauth", func(t *testing.T) {
		update := &service.UpdateStoryDto{
			Name:             story.Name,
			Description:      "This is a better description",
			Keywords:         story.Keywords,
			TeamkatalogenURL: story.TeamkatalogenURL,
			ProductAreaID:    &pa1,
			TeamID:           &nada,
			Group:            story.Group,
		}

		story.Description = update.Description

		got := &service.Story{}

		NewTester(t, server).
			Put(update, "/api/stories/"+story.ID.String()).
			HasStatusCode(http.StatusOK).
			Expect(story, got, cmpopts.IgnoreFields(service.Story{}, "LastModified"))

		story = got
	})

	t.Run("Get story with oauth", func(t *testing.T) {
		got := &service.Story{}

		NewTester(t, server).
			Get("/api/stories/"+story.ID.String()).
			HasStatusCode(http.StatusOK).
			Expect(story, got)
	})

	t.Run("Get index", func(t *testing.T) {
		data := NewTester(t, server).
			Get("/story/" + story.ID.String()).
			HasStatusCode(http.StatusOK).
			Body()

		assert.Equal(t, defaultHtml, data)
	})

	t.Run("Delete story with oauth", func(t *testing.T) {
		NewTester(t, server).
			Delete("/api/stories/" + story.ID.String()).
			HasStatusCode(http.StatusOK)
	})

	t.Run("Get story with oauth after delete", func(t *testing.T) {
		NewTester(t, server).
			Get("/api/stories/" + story.ID.String()).
			HasStatusCode(http.StatusNotFound)
	})

	token, err := tokenStorage.GetNadaToken(context.Background(), "nada")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Create story without token", func(t *testing.T) {
		NewTester(t, server).
			Post(&service.NewStory{}, "/story/create").
			HasStatusCode(http.StatusUnauthorized)
	})

	t.Run("Create story for team with token", func(t *testing.T) {
		newStory := &service.NewStory{
			Name:          "My new story",
			Description:   strToStrPtr("This is my story, and it is pretty bad"),
			Keywords:      []string{"story", "bad"},
			ProductAreaID: &pa1,
			TeamID:        &nada,
			Group:         "nada@nav.no",
		}

		expect := &service.Story{
			Name:             "My new story",
			Creator:          "nada@nav.no",
			Description:      "This is my story, and it is pretty bad",
			Keywords:         []string{"story", "bad"},
			TeamkatalogenURL: strToStrPtr("http://example.com/team/00000000-0000-4444-0000-000000000000"),
			TeamID:           &nada,
			Group:            "nada@nav.no",
			// FIXME: can't set these from CreateStory, should they be?
			// TeamName:         strToStrPtr("Team1"),
			// ProductAreaName:  "Product area 1",
		}

		NewTester(t, server).
			Headers(map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token)}).
			Post(newStory, "/story/create").
			HasStatusCode(http.StatusOK).
			Expect(expect, story, cmpopts.IgnoreFields(service.Story{}, "ID", "Created", "LastModified"))
	})

	t.Run("Recreate story files with token", func(t *testing.T) {
		files := map[string]string{
			"index.html":                   defaultHtml,
			"subpage/index.html":           "<html><h1>Subpage</h1></html>",
			"subsubsubpage/something.html": "<html><h1>Subsubsubpage</h1></html>",
		}

		req := CreateMultipartFormRequest(
			t,
			http.MethodPut,
			server.URL+"/story/update/"+story.ID.String(),
			files,
			nil,
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", token),
			},
		)

		NewTester(t, server).
			Send(req).
			HasStatusCode(http.StatusNoContent)

		for path, content := range files {
			got := NewTester(t, server).
				Get("/story/" + story.ID.String() + "/" + path).
				HasStatusCode(http.StatusOK).
				Body()

			assert.Equal(t, content, got)
		}
	})

	t.Run("Recreate story files with token and nais-team prefix", func(t *testing.T) {
		storage := postgres.NewStoryStorage(repo)

		updateStory, err := storage.CreateStory(context.Background(), "nais-team-nada@nav.no", &service.NewStory{
			Name:        "My update story",
			Description: strToStrPtr("This is my update story, and it is pretty bad"),
			Keywords:    []string{"story", "bad"},
			Group:       "nais-team-nada@nav.no",
		})
		assert.NoError(t, err)

		files := map[string]string{
			"index.html":                   defaultHtml,
			"subpage/index.html":           "<html><h1>Subpage</h1></html>",
			"subsubsubpage/something.html": "<html><h1>Subsubsubpage</h1></html>",
		}

		req := CreateMultipartFormRequest(
			t,
			http.MethodPut,
			server.URL+"/quarto/update/"+updateStory.ID.String(),
			files,
			nil,
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", token),
			},
		)

		NewTester(t, server).
			Send(req).
			HasStatusCode(http.StatusNoContent)

		for path, content := range files {
			got := NewTester(t, server).
				Get("/quarto/" + updateStory.ID.String() + "/" + path).
				HasStatusCode(http.StatusOK).
				Body()

			assert.Equal(t, content, got)
		}
	})

	t.Run("Append story files with token", func(t *testing.T) {
		files := map[string]string{
			"newpage/test.html": "<html><h1>New page</h1></html>",
		}

		req := CreateMultipartFormRequest(
			t,
			http.MethodPatch,
			server.URL+"/story/update/"+story.ID.String(),
			files,
			nil,
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", token),
			},
		)

		NewTester(t, server).
			Send(req).
			HasStatusCode(http.StatusNoContent)

		got := NewTester(t, server).
			Get("/story/" + story.ID.String() + "/newpage/test.html").
			HasStatusCode(http.StatusOK).
			Body()

		assert.Equal(t, "<html><h1>New page</h1></html>", got)
	})
}
