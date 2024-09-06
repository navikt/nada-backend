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

func TestUserDataService(t *testing.T) {
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
	StorageCreateNaisConsoleTeamsAndProjects(t, stores.NaisConsoleStorage, map[string]string{GroupNameNada: GroupNameNada, GroupNameReef: GroupEmailReef})
	fuel := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductBiofuelProduction(GroupEmailNada, TeamSeagrassID))
	barriers := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductProtectiveBarriers(GroupEmailReef, TeamReefID))
	feed := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductAquacultureFeed(GroupEmailNada, TeamSeagrassID))
	reef := StorageCreateDataproduct(t, stores.DataProductsStorage, NewDataProductReefMonitoring(GroupEmailReef, TeamReefID))

	user := &service.User{
		Email: "bob.the.builder@example.com",
		GoogleGroups: []service.Group{
			{
				Name:  GroupNameNada,
				Email: GroupEmailNada,
			},
			{
				Name:  GroupNameReef,
				Email: GroupEmailReef,
			},
		},
	}
	fuelInsights := StorageCreateInsightProduct(t, user.Email, stores.InsightProductStorage, NewInsightProductBiofuelProduction(GroupEmailNada, TeamSeagrassID))
	barriersInsights := StorageCreateInsightProduct(t, user.Email, stores.InsightProductStorage, NewInsightProductProtectiveBarriers(GroupEmailReef, TeamReefID))
	feedInsights := StorageCreateInsightProduct(t, user.Email, stores.InsightProductStorage, NewInsightProductAquacultureFeed(GroupEmailNada, TeamSeagrassID))
	reefInsights := StorageCreateInsightProduct(t, user.Email, stores.InsightProductStorage, NewInsightProductReefMonitoring(GroupEmailReef, TeamReefID))

	fuelStory := StorageCreateStory(t, stores.StoryStorage, user.Email, NewStoryBiofuelProduction(GroupEmailNada))
	barriersStory := StorageCreateStory(t, stores.StoryStorage, user.Email, NewStoryProtectiveBarriers(GroupEmailReef))
	feedStory := StorageCreateStory(t, stores.StoryStorage, user.Email, NewStoryAquacultureFeed(GroupEmailNada))
	reefStory := StorageCreateStory(t, stores.StoryStorage, user.Email, NewStoryReefMonitoring(GroupEmailReef))

	{
		s := core.NewUserService(stores.AccessStorage, stores.TokenStorage, stores.StoryStorage, stores.DataProductsStorage,
			stores.InsightProductStorage, stores.NaisConsoleStorage, log)
		h := handlers.NewUserHandler(s)
		e := routes.NewUserEndpoints(log, h)
		f := routes.NewUserRoutes(e, injectUser(user))
		f(r)
	}

	server := httptest.NewServer(r)
	defer server.Close()
	// Would prefer to sort by dp.team_name, but it is always null
	t.Run("User data products are sorted alphabetically by group_name and dp_name", func(t *testing.T) {
		got := &service.UserInfo{}
		expect := []service.Dataproduct{
			{ID: feed.ID, Name: feed.Name, Owner: &service.DataproductOwner{Group: GroupEmailNada}},
			{ID: fuel.ID, Name: fuel.Name, Owner: &service.DataproductOwner{Group: GroupEmailNada}},
			{ID: barriers.ID, Name: barriers.Name, Owner: &service.DataproductOwner{Group: GroupEmailReef}},
			{ID: reef.ID, Name: reef.Name, Owner: &service.DataproductOwner{Group: GroupEmailReef}},
		}

		NewTester(t, server).Get("/api/userData").
			HasStatusCode(http.StatusOK).Value(got)

		if len(got.Dataproducts) != len(expect) {
			t.Fatalf("got %d, expected %d", len(got.Dataproducts), len(expect))
		}
		for i := 0; i < len(got.Dataproducts); i++ {
			if got.Dataproducts[i].ID != expect[i].ID {
				t.Errorf("got %s, expected %s", got.Dataproducts[i].ID, expect[i].ID)
			}
			if got.Dataproducts[i].Name != expect[i].Name {
				t.Errorf("got %s, expected %s", got.Dataproducts[i].Name, expect[i].Name)
			}
			if got.Dataproducts[i].Owner.Group != expect[i].Owner.Group {
				t.Errorf("got %s, expected %s", got.Dataproducts[i].Owner.Group, expect[i].Owner.Group)
			}
		}
	})
	// Would prefer to sort by team_name, but it is null in the view
	t.Run("User insight products are sorted alphabetically by group and name", func(t *testing.T) {
		got := &service.UserInfo{}
		expect := []service.InsightProduct{
			{ID: feedInsights.ID, Name: feedInsights.Name, Group: GroupEmailNada},
			{ID: fuelInsights.ID, Name: fuelInsights.Name, Group: GroupEmailNada},
			{ID: barriersInsights.ID, Name: barriersInsights.Name, Group: GroupEmailReef},
			{ID: reefInsights.ID, Name: reefInsights.Name, Group: GroupEmailReef},
		}

		NewTester(t, server).Get("/api/userData").
			HasStatusCode(http.StatusOK).Value(got)

		if len(got.InsightProducts) != len(expect) {
			t.Fatalf("got %d, expected %d", len(got.InsightProducts), len(expect))
		}
		for i := 0; i < len(got.InsightProducts); i++ {
			if got.InsightProducts[i].ID != expect[i].ID {
				t.Errorf("got %s, expected %s", got.InsightProducts[i].ID, expect[i].ID)
			}
			if got.InsightProducts[i].Name != expect[i].Name {
				t.Errorf("got %s, expected %s", got.InsightProducts[i].Name, expect[i].Name)
			}
			if got.InsightProducts[i].Group != expect[i].Group {
				t.Errorf("got %s, expected %s", got.InsightProducts[i].Group, expect[i].Group)
			}
		}
	})

	t.Run("User stories are sorted alphabetically by group_name and name", func(t *testing.T) {
		got := &service.UserInfo{}
		expect := []service.Story{
			{ID: feedStory.ID, Name: feedStory.Name, Group: GroupEmailNada},
			{ID: fuelStory.ID, Name: fuelStory.Name, Group: GroupEmailNada},
			{ID: barriersStory.ID, Name: barriersStory.Name, Group: GroupEmailReef},
			{ID: reefStory.ID, Name: reefStory.Name, Group: GroupEmailReef},
		}

		NewTester(t, server).Get("/api/userData").
			HasStatusCode(http.StatusOK).Value(got)

		if len(got.Stories) != len(expect) {
			t.Fatalf("got %d, expected %d", len(got.Stories), len(expect))
		}
		for i := 0; i < len(got.Stories); i++ {
			if got.Stories[i].ID != expect[i].ID {
				t.Errorf("got %s, expected %s", got.Stories[i].ID, expect[i].ID)
			}
			if got.Stories[i].Name != expect[i].Name {
				t.Errorf("got %s, expected %s", got.Stories[i].Name, expect[i].Name)
			}
			if got.Stories[i].Group != expect[i].Group {
				t.Errorf("got %s, expected %s", got.Stories[i].Group, expect[i].Group)
			}
		}
	})
}
