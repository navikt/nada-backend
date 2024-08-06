package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type InsightProductEndpoints struct {
	GetInsightProduct    http.HandlerFunc
	CreateInsightProduct http.HandlerFunc
	UpdateInsightProduct http.HandlerFunc
	DeleteInsightProduct http.HandlerFunc
}

func NewInsightProductEndpoints(log zerolog.Logger, h *handlers.InsightProductHandler) *InsightProductEndpoints {
	return &InsightProductEndpoints{
		GetInsightProduct:    transport.For(h.GetInsightProduct).Build(log),
		CreateInsightProduct: transport.For(h.CreateInsightProduct).RequestFromJSON().Build(log),
		UpdateInsightProduct: transport.For(h.UpdateInsightProduct).RequestFromJSON().Build(log),
		DeleteInsightProduct: transport.For(h.DeleteInsightProduct).Build(log),
	}
}

func NewInsightProductRoutes(endpoints *InsightProductEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/insightProducts", func(r chi.Router) {
			r.Use(auth)
			r.Get("/{id}", endpoints.GetInsightProduct)
			r.Post("/new", endpoints.CreateInsightProduct)
			r.Put("/{id}", endpoints.UpdateInsightProduct)
			r.Delete("/{id}", endpoints.DeleteInsightProduct)
		})
	}
}
