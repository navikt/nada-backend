package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/sirupsen/logrus"
)

type MockHTTP struct {
	repo *database.Repo
	log  *logrus.Entry
}

func NewMockHTTP(repo *database.Repo, log *logrus.Entry) *MockHTTP {
	return &MockHTTP{
		log:  log,
		repo: repo,
	}
}

func (h MockHTTP) Login(w http.ResponseWriter, r *http.Request) {
	loginPage := r.URL.Query().Get("redirect_uri")

	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000" + loginPage
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "token",
		Path:     "/",
		Domain:   "localhost",
		Expires:  time.Now().Add(time.Hour * 24),
		Secure:   true,
		HttpOnly: true,
	})

	http.Redirect(w, r, loginPage, http.StatusFound)
}

func (h *MockHTTP) Callback(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

func (h *MockHTTP) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Domain:   "localhost",
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
	})

	var loginPage string
	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = "/"
	}

	http.Redirect(w, r, loginPage, http.StatusFound)
}

func (h *MockHTTP) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("jwt")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), auth.ContextUserKey, &auth.MockUser)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
