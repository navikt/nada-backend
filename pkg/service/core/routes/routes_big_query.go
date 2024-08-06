package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type BigQueryEndpoints struct {
	GetBigQueryColumns  http.HandlerFunc
	GetBigQueryTables   http.HandlerFunc
	GetBigQueryDatasets http.HandlerFunc
	SyncBigQueryTables  http.HandlerFunc
}

func NewBigQueryEndpoints(log zerolog.Logger, h *handlers.BigQueryHandler) *BigQueryEndpoints {
	return &BigQueryEndpoints{
		GetBigQueryColumns:  transport.For(h.GetBigQueryColumns).Build(log),
		GetBigQueryTables:   transport.For(h.GetBigQueryTables).Build(log),
		GetBigQueryDatasets: transport.For(h.GetBigQueryDatasets).Build(log),
		SyncBigQueryTables:  transport.For(h.SyncBigQueryTables).Build(log),
	}
}

func NewBigQueryRoutes(endpoints *BigQueryEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/bigquery/columns", func(r chi.Router) {
			r.Get("/", endpoints.GetBigQueryColumns)
		})

		router.Route("/api/bigquery/tables", func(r chi.Router) {
			r.Get("/", endpoints.GetBigQueryTables)
		})

		router.Route("/api/bigquery/datasets", func(r chi.Router) {
			r.Get("/", endpoints.GetBigQueryDatasets)
		})

		router.Route("/api/bigquery/tables/sync", func(r chi.Router) {
			r.Post("/", endpoints.SyncBigQueryTables)
		})
	}
}
