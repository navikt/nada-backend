package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type SearchEndpoints struct {
	Search http.HandlerFunc
}

func NewSearchEndpoints(log zerolog.Logger, h *handlers.SearchHandler) *SearchEndpoints {
	return &SearchEndpoints{
		Search: transport.For(h.Search).Build(log),
	}
}

func NewSearchRoutes(endpoints *SearchEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/search", func(r chi.Router) {
			r.Get("/", endpoints.Search)
		})
	}
}
