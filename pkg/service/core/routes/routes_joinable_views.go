package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type JoinableViewsEndpoints struct {
	CreateJoinableViews     http.HandlerFunc
	GetJoinableViewsForUser http.HandlerFunc
	GetJoinableView         http.HandlerFunc
}

func NewJoinableViewsEndpoints(log zerolog.Logger, h *handlers.JoinableViewsHandler) *JoinableViewsEndpoints {
	return &JoinableViewsEndpoints{
		CreateJoinableViews:     transport.For(h.CreateJoinableViews).RequestFromJSON().Build(log),
		GetJoinableViewsForUser: transport.For(h.GetJoinableViewsForUser).Build(log),
		GetJoinableView:         transport.For(h.GetJoinableView).Build(log),
	}
}

func NewJoinableViewsRoutes(endpoints *JoinableViewsEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/pseudo/joinable", func(r chi.Router) {
			r.Use(auth)
			r.Post("/new", endpoints.CreateJoinableViews)
			r.Get("/", endpoints.GetJoinableViewsForUser)
			r.Get("/{id}", endpoints.GetJoinableView)
		})
	}
}
