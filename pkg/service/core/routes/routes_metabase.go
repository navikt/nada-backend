package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type MetabaseEndpoints struct {
	MapDataset http.HandlerFunc
}

func NewMetabaseEndpoints(log zerolog.Logger, h *handlers.MetabaseHandler) *MetabaseEndpoints {
	return &MetabaseEndpoints{
		MapDataset: transport.For(h.MapDataset).RequestFromJSON().Build(log),
	}
}

func NewMetabaseRoutes(endpoints *MetabaseEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		// Might otherwise conflict with DatasetRoutes in routes_dataproducts.go
		router.With(auth).Post("/api/datasets/{id}/map", endpoints.MapDataset)
	}
}
