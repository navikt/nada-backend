package routes

import (
	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"net/http"
)

type ProductAreaEndpoints struct {
	GetProductAreas          http.HandlerFunc
	GetProductAreaWithAssets http.HandlerFunc
}

func NewProductAreaEndpoints(log zerolog.Logger, h *handlers.ProductAreasHandler) *ProductAreaEndpoints {
	return &ProductAreaEndpoints{
		GetProductAreas:          transport.For(h.GetProductAreas).Build(log),
		GetProductAreaWithAssets: transport.For(h.GetProductAreaWithAssets).Build(log),
	}
}

func NewProductAreaRoutes(endpoints *ProductAreaEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/productareas", func(r chi.Router) {
			r.Get("/", endpoints.GetProductAreas)
			r.Get("/{id}", endpoints.GetProductAreaWithAssets)
		})
	}
}
