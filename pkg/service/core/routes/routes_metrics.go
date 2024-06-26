package routes

import (
	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type MetricsEndpoints struct {
	GetMetrics http.Handler
}

func NewMetricsEndpoints(promReg *prometheus.Registry) *MetricsEndpoints {
	return &MetricsEndpoints{
		GetMetrics: promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}),
	}
}

func NewMetricsRoutes(endpoints *MetricsEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Handle("/internal/metrics", endpoints.GetMetrics)
	}
}
