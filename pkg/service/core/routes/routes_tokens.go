package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type TokensEndpoints struct {
	GetAllTeamTokens http.HandlerFunc
	RotateNadaToken  http.HandlerFunc
}

func NewTokensEndpoints(log zerolog.Logger, h *handlers.TokenHandler) *TokensEndpoints {
	return &TokensEndpoints{
		GetAllTeamTokens: h.GetAllTeamTokens,
		RotateNadaToken:  transport.For(h.RotateNadaToken).Build(log),
	}
}

func NewTokensRoutes(endpoints *TokensEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/user", func(r chi.Router) {
			r.Use(auth)
			r.Put("/token", endpoints.RotateNadaToken)
		})

		router.Get("/internal/teamtokens", endpoints.GetAllTeamTokens)
	}
}
