package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type KeywordEndpoints struct {
	GetKeywordsListSortedByPopularity http.HandlerFunc
	UpdateKeywords                    http.HandlerFunc
}

func NewKeywordEndpoints(log zerolog.Logger, h *handlers.KeywordsHandler) *KeywordEndpoints {
	return &KeywordEndpoints{
		GetKeywordsListSortedByPopularity: transport.For(h.GetKeywordsListSortedByPopularity).Build(log),
		UpdateKeywords:                    transport.For(h.UpdateKeywords).RequestFromJSON().Build(log),
	}
}

func NewKeywordRoutes(endpoints *KeywordEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/keywords", func(r chi.Router) {
			r.Use(auth)
			r.Get("/", endpoints.GetKeywordsListSortedByPopularity)
			r.Post("/", endpoints.UpdateKeywords)
		})
	}
}
