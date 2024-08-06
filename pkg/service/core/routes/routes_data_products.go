package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type DataProductsEndpoints struct {
	GetDataProduct                     http.HandlerFunc
	CreateDataProduct                  http.HandlerFunc
	DeleteDataProduct                  http.HandlerFunc
	UpdateDataProduct                  http.HandlerFunc
	GetDatasetsMinimal                 http.HandlerFunc
	GetDataset                         http.HandlerFunc
	CreateDataset                      http.HandlerFunc
	UpdateDataset                      http.HandlerFunc
	DeleteDataset                      http.HandlerFunc
	GetAccessiblePseudoDatasetsForUser http.HandlerFunc
}

func NewDataProductsEndpoints(log zerolog.Logger, h *handlers.DataProductsHandler) *DataProductsEndpoints {
	return &DataProductsEndpoints{
		GetDataProduct:                     transport.For(h.GetDataProduct).Build(log),
		CreateDataProduct:                  transport.For(h.CreateDataProduct).RequestFromJSON().Build(log),
		DeleteDataProduct:                  transport.For(h.DeleteDataProduct).Build(log),
		UpdateDataProduct:                  transport.For(h.UpdateDataProduct).RequestFromJSON().Build(log),
		GetDatasetsMinimal:                 transport.For(h.GetDatasetsMinimal).Build(log),
		GetDataset:                         transport.For(h.GetDataset).Build(log),
		CreateDataset:                      transport.For(h.CreateDataset).RequestFromJSON().Build(log),
		UpdateDataset:                      transport.For(h.UpdateDataset).RequestFromJSON().Build(log),
		DeleteDataset:                      transport.For(h.DeleteDataset).Build(log),
		GetAccessiblePseudoDatasetsForUser: transport.For(h.GetAccessiblePseudoDatasetsForUser).Build(log),
	}
}

func NewDataProductsRoutes(endpoints *DataProductsEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/dataproducts", func(r chi.Router) {
			r.Use(auth)
			r.Get("/{id}", endpoints.GetDataProduct)
			r.Post("/new", endpoints.CreateDataProduct)
			r.Delete("/{id}", endpoints.DeleteDataProduct)
			r.Put("/{id}", endpoints.UpdateDataProduct)
		})

		// Might otherwise conflict with MetabaseRoutes in routes_metabase.go
		router.With(auth).Get("/api/datasets/", endpoints.GetDatasetsMinimal)
		router.With(auth).Get("/api/datasets/{id}", endpoints.GetDataset)
		router.With(auth).Post("/api/datasets/new", endpoints.CreateDataset)
		router.With(auth).Put("/api/datasets/{id}", endpoints.UpdateDataset)
		router.With(auth).Delete("/api/datasets/{id}", endpoints.DeleteDataset)
		router.With(auth).Get("/api/datasets/pseudo/accessible", endpoints.GetAccessiblePseudoDatasetsForUser)
	}
}
