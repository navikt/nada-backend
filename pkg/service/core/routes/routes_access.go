package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type AccessEndpoints struct {
	GetAccessRequests     http.HandlerFunc
	ProcessAccessRequest  http.HandlerFunc
	CreateAccessRequest   http.HandlerFunc
	DeleteAccessRequest   http.HandlerFunc
	UpdateAccessRequest   http.HandlerFunc
	GrantAccessToDataset  http.HandlerFunc
	RevokeAccessToDataset http.HandlerFunc
}

func NewAccessEndpoints(log zerolog.Logger, h *handlers.AccessHandler) *AccessEndpoints {
	return &AccessEndpoints{
		GetAccessRequests:     transport.For(h.GetAccessRequests).Build(log),
		ProcessAccessRequest:  transport.For(h.ProcessAccessRequest).Build(log),
		CreateAccessRequest:   transport.For(h.NewAccessRequest).RequestFromJSON().Build(log),
		DeleteAccessRequest:   transport.For(h.DeleteAccessRequest).Build(log),
		UpdateAccessRequest:   transport.For(h.UpdateAccessRequest).RequestFromJSON().Build(log),
		GrantAccessToDataset:  transport.For(h.GrantAccessToDataset).RequestFromJSON().Build(log),
		RevokeAccessToDataset: transport.For(h.RevokeAccessToDataset).Build(log),
	}
}

func NewAccessRoutes(endpoints *AccessEndpoints, auth func(http.Handler) http.Handler) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api/accessRequests", func(r chi.Router) {
			r.Use(auth)
			r.Get("/", endpoints.GetAccessRequests)
			r.Post("/process/{id}", endpoints.ProcessAccessRequest)
			r.Post("/new", endpoints.CreateAccessRequest)
			r.Delete("/{id}", endpoints.DeleteAccessRequest)
			// FIXME: dont seem to use the ID in the URL
			r.Put("/{id}", endpoints.UpdateAccessRequest)
		})

		router.Route("/api/accesses", func(r chi.Router) {
			r.Use(auth)
			r.Post("/grant", endpoints.GrantAccessToDataset)
			r.Post("/revoke", endpoints.RevokeAccessToDataset)
		})
	}
}
