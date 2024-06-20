package api

import (
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type HTTPAPI interface {
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

func New(
	endpoints *handlers.Endpoints,
	realHandlers *handlers.Handlers,
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	promReg *prometheus.Registry,
) *chi.Mux {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	router := chi.NewRouter()

	router.Use(corsMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
	})

	router.Route(`/{story|quarto}/`, func(r chi.Router) {
		// FIXME: not great, move into middlewares something
		r.Use(handlers.StoryHTTPMiddleware(realHandlers.StoryHandler))
		r.Get("/*", endpoints.GetGCSObject)
		r.Post("/create", endpoints.CreateStoryHTTP)
		r.Route("/update/{id}", func(r chi.Router) {
			r.Put("/", endpoints.UpdateStoryHTTP)
			r.Patch("/", endpoints.AppendStoryHTTP)
		})
	})

	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
		r.Get("/teamtokens", endpoints.GetAllTeamTokens)
	})

	router.Route("/api/dataproducts", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/{id}", endpoints.GetDataProduct)
		r.Post("/new", endpoints.CreateDataProduct)
		r.Delete("/{id}", endpoints.DeleteDataProduct)
		r.Put("/{id}", endpoints.UpdateDataProduct)
	})

	router.Route("/api/datasets", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", endpoints.GetDatasetsMinimal)
		r.Get("/{id}", endpoints.GetDataset)
		r.Post("/{id}/map", endpoints.MapDataset)
		r.Post("/new", endpoints.CreateDataset)
		r.Put("/{id}", endpoints.UpdateDataset)
		r.Delete("/{id}", endpoints.DeleteDataset)
		r.Get("/pseudo/accessible", endpoints.GetAccessiblePseudoDatasetsForUser)
	})

	router.Route("/api/accessRequests", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", endpoints.GetAccessRequests)
		r.Post("/process/{id}", endpoints.ProcessAccessRequest)
		r.Post("/new", endpoints.CreateAccessRequest)
		r.Delete("/{id}", endpoints.DeleteAccessRequest)
		// FIXME: dont seem to use the ID in the URL
		r.Put("/{id}", endpoints.UpdateAccessRequest)
	})

	router.Route("/api/polly", func(r chi.Router) {
		r.Get("/", endpoints.SearchPolly)
	})

	router.Route("/api/productareas", func(r chi.Router) {
		r.Get("/", endpoints.GetProductAreas)
		r.Get("/{id}", endpoints.GetProductAreaWithAssets)
	})

	router.Route("/api/teamkatalogen", func(r chi.Router) {
		r.Get("/", endpoints.SearchTeamKatalogen)
	})

	router.Route("/api/keywords", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", endpoints.GetKeywordsListSortedByPopularity)
		r.Post("/", endpoints.UpdateKeywords)
	})

	router.Route("/api/bigquery/columns", func(r chi.Router) {
		r.Get("/", endpoints.GetBigQueryColumns)
	})

	router.Route("/api/bigquery/tables", func(r chi.Router) {
		r.Get("/", endpoints.GetBigQueryTables)
	})

	router.Route("/api/bigquery/datasets", func(r chi.Router) {
		r.Get("/", endpoints.GetBigQueryDatasets)
	})

	router.Route("/api/bigquery/tables/sync", func(r chi.Router) {
		r.Post("/", endpoints.SyncBigQueryTables)
	})

	router.Route("/api/search", func(r chi.Router) {
		r.Get("/", endpoints.Search)
	})

	router.Route("/api/user", func(r chi.Router) {
		r.Use(authMW)
		r.Put("/token", endpoints.RotateNadaToken)
	})

	router.Route("/api/userData", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", endpoints.GetUserData)
	})

	router.Route("/api/slack", func(r chi.Router) {
		r.Get("/isValid", endpoints.IsValidSlackChannel)
	})

	router.Route("/api/stories", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/{id}", endpoints.GetStoryMetadata)
		r.Post("/new", endpoints.CreateStory)
		r.Put("/{id}", endpoints.UpdateStory)
		r.Delete("/{id}", endpoints.DeleteStory)
	})

	router.Route("/api/accesses", func(r chi.Router) {
		r.Use(authMW)
		r.Post("/grant", endpoints.GrantAccessToDataset)
		r.Post("/revoke", endpoints.RevokeAccessToDataset)
	})

	router.Route("/api/pseudo/joinable", func(r chi.Router) {
		r.Use(authMW)
		r.Post("/new", endpoints.CreateJoinableViews)
		r.Get("/", endpoints.GetJoinableViewsForUser)
		r.Get("/{id}", endpoints.GetJoinableView)
	})

	router.Route("/api/insightProducts", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/{id}", endpoints.GetInsightProduct)
		r.Post("/new", endpoints.CreateInsightProduct)
		r.Put("/{id}", endpoints.UpdateInsightProduct)
		r.Delete("/{id}", endpoints.DeleteInsightProduct)
	})

	return router
}
