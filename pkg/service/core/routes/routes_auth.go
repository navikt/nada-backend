package routes

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/api"
)

type AuthEndpoints struct {
	Login    http.HandlerFunc
	Callback http.HandlerFunc
	Logout   http.HandlerFunc
}

func NewAuthEndpoints(api api.HTTP) *AuthEndpoints {
	return &AuthEndpoints{
		Login:    api.Login,
		Callback: api.Callback,
		Logout:   api.Logout,
	}
}

func NewAuthRoutes(endpoints *AuthEndpoints) AddRoutesFn {
	return func(router chi.Router) {
		router.Route("/api", func(r chi.Router) {
			r.HandleFunc("/login", endpoints.Login)
			r.HandleFunc("/oauth2/callback", endpoints.Callback)
			r.HandleFunc("/logout", endpoints.Logout)
		})
	}
}
