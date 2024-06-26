package routes

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
)

type AddRoutesFn func(router chi.Router)

func Add(r chi.Router, routes ...AddRoutesFn) {
	cors := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	r.Use(cors)

	for _, route := range routes {
		route(r)
	}
}
