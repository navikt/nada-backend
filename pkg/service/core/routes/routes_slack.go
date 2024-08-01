package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type SlackEndpoints struct {
	IsValidSlackChannel http.HandlerFunc
}

func NewSlackEndpoints(log zerolog.Logger, h *handlers.SlackHandler) *SlackEndpoints {
	return &SlackEndpoints{
		IsValidSlackChannel: transport.For(h.IsValidSlackChannel).Build(log),
	}
}

func NewSlackRoutes(endpoints *SlackEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/slack", func(r chi.Router) {
			r.Get("/isValid", endpoints.IsValidSlackChannel)
		})
	}
}
