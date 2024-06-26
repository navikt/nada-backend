package routes

import (
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"net/http"
)

type PollyEndpoints struct {
	SearchPolly http.HandlerFunc
}

func NewPollyEndpoints(log zerolog.Logger, h *handlers.PollyHandler) *PollyEndpoints {
	return &PollyEndpoints{
		SearchPolly: transport.For(h.SearchPolly).Build(log),
	}
}

func NewPollyRoutes(endpoints *PollyEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/polly", func(r chi.Router) {
			r.Get("/", endpoints.SearchPolly)
		})
	}
}
