package routes

import (
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"net/http"
)

type StoryEndpoints struct {
	GetGCSObject     http.HandlerFunc
	CreateStoryHTTP  http.HandlerFunc
	UpdateStoryHTTP  http.HandlerFunc
	AppendStoryHTTP  http.HandlerFunc
	GetStoryMetadata http.HandlerFunc
	CreateStory      http.HandlerFunc
	UpdateStory      http.HandlerFunc
	DeleteStory      http.HandlerFunc
	StoryMiddleware  func(http.Handler) http.Handler
}

func NewStoryEndpoints(log zerolog.Logger, h *handlers.StoryHandler) *StoryEndpoints {
	return &StoryEndpoints{
		GetGCSObject:     h.GetGCSObject,
		CreateStoryHTTP:  h.CreateStoryHTTP,
		UpdateStoryHTTP:  h.UpdateStoryHTTP,
		AppendStoryHTTP:  h.AppendStoryHTTP,
		GetStoryMetadata: transport.For(h.GetStoryMetadata).Build(log),
		CreateStory:      transport.For(h.CreateStory).Build(log),
		UpdateStory:      transport.For(h.UpdateStory).RequestFromJSON().Build(log),
		DeleteStory:      transport.For(h.DeleteStory).Build(log),
		StoryMiddleware:  handlers.StoryHTTPMiddleware(h),
	}
}

func NewStoryRoutes(endpoints *StoryEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route(`/{story|quarto}/`, func(r chi.Router) {
			// FIXME: I don't get why we need this
			r.Use(endpoints.StoryMiddleware)
			r.Get("/*", endpoints.GetGCSObject)
			r.Post("/create", endpoints.CreateStoryHTTP)
			// FIXME: Shouldn't these have auth?
			r.Route("/update/{id}", func(r chi.Router) {
				r.Put("/", endpoints.UpdateStoryHTTP)
				r.Patch("/", endpoints.AppendStoryHTTP)
			})
		})

		router.Route("/api/stories", func(r chi.Router) {
			r.Use(auth)
			r.Get("/{id}", endpoints.GetStoryMetadata)
			r.Post("/new", endpoints.CreateStory)
			r.Put("/{id}", endpoints.UpdateStory)
			r.Delete("/{id}", endpoints.DeleteStory)
		})
	}
}
