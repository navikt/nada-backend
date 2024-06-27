package integration

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestKeywords(t *testing.T) {
	c := NewContainers(t)
	defer c.Cleanup()

	pgCfg := c.RunPostgres(NewPostgresConfig())

	repo, err := database.New(
		pgCfg.ConnectionURL(),
		10,
		10,
		log.WithField("subsystem", "repo"),
	)
	assert.NoError(t, err)

	r := chi.NewRouter()

	storyStorage := postgres.NewStoryStorage(repo)
	_, err = storyStorage.CreateStory(context.Background(), "bob@example.com", &service.NewStory{
		Name:     "A story",
		Keywords: []string{"keyword1", "keyword2", "keyword3"},
		Group:    "nada@nav.no",
	})
	assert.NoError(t, err)

	{
		store := postgres.NewKeywordsStorage(repo)
		s := core.NewKeywordsService(store, "nada@nav.no")
		h := handlers.NewKeywordsHandler(s)
		e := routes.NewKeywordEndpoints(zerolog.New(os.Stdout), h)
		f := routes.NewKeywordRoutes(e, injectUser(&service.User{
			Email: "bob@example.com",
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

	t.Run("Update keywords", func(t *testing.T) {

		NewTester(t, server).
			Post("/api/keywords", &service.UpdateKeywordsDto{
				ObsoleteKeywords: []string{"keyword1"},
				ReplacedKeywords: []string{"keyword2"},
				NewText:          []string{"keyword2_replaced"},
			}).HasStatusCode(http.StatusNoContent)
	})

	t.Run("Get keywords", func(t *testing.T) {
		into := &service.KeywordsList{}

		expect := &service.KeywordsList{
			KeywordItems: []service.KeywordItem{
				{
					Keyword: "keyword3",
					Count:   1,
				},
				{
					Keyword: "keyword2_replaced",
					Count:   1,
				},
			},
		}

		NewTester(t, server).
			Get("/api/keywords").
			HasStatusCode(http.StatusOK).
			Expect(expect, into)
	})
}
