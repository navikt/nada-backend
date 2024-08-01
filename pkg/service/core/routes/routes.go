package routes

import (
	"fmt"
	"io"
	"net/http"

	"github.com/docker/cli/cli/command/formatter/tabwriter"
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

func Print(r chi.Router, out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "Method\tRoute\tMiddlewares")

	err := chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\n", method, route, len(middlewares))

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking routes: %w", err)
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}
