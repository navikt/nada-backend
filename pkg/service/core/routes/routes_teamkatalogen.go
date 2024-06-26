package routes

import (
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"net/http"
)

type TeamkatalogenEndpoints struct {
	SearchTeamKatalogen http.HandlerFunc
}

func NewTeamkatalogenEndpoints(log zerolog.Logger, h *handlers.TeamkatalogenHandler) *TeamkatalogenEndpoints {
	return &TeamkatalogenEndpoints{
		SearchTeamKatalogen: transport.For(h.SearchTeamKatalogen).Build(log),
	}
}

func NewTeamkatalogenRoutes(endpoints *TeamkatalogenEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/teamkatalogen", func(r chi.Router) {
			r.Get("/", endpoints.SearchTeamKatalogen)
		})
	}
}
