package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
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

	session := &models.Session{
		Token:   generateSecureToken(tokenLength),
		Expires: time.Now().Add(sessionLength),
		Email:   auth.MockUser.Email,
		Name:    auth.MockUser.Name,
	}

	if err := h.repo.CreateSession(r.Context(), session); err != nil {
		h.log.WithError(err).Error("Unable to store session")
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	// TODO(thokra): Encrypt cookie value
	http.SetCookie(w, &http.Cookie{
		Name:     "nada_session",
		Value:    session.Token,
		Path:     "/",
		Domain:   r.Host,
		Expires:  session.Expires,
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
		Name:     "nada_backend",
		Value:    "",
		Path:     "/",
		Domain:   r.Host,
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
