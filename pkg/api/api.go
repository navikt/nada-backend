package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	RedirectURICookie               = "redirecturi"
	OAuthStateCookie                = "oauthstate"
	tokenLength                     = 32
	sessionLength     time.Duration = 7 * time.Hour
)

type OAuth2 interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

type HTTP struct {
	oauth2Config OAuth2
	log          *logrus.Entry
	repo         *database.Repo
}

func NewHTTP(oauth2Config OAuth2, repo *database.Repo, log *logrus.Entry) HTTP {
	return HTTP{
		oauth2Config: oauth2Config,
		log:          log,
		repo:         repo,
	}
}

func (h HTTP) Login(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	http.SetCookie(w, &http.Cookie{
		Name:     RedirectURICookie,
		Value:    redirectURI,
		Path:     "/",
		Domain:   r.Host,
		Expires:  time.Now().Add(30 * time.Minute),
		Secure:   true,
		HttpOnly: true,
	})

	oauthState := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookie,
		Value:    oauthState,
		Path:     "/",
		Domain:   r.Host,
		Expires:  time.Now().Add(30 * time.Minute),
		Secure:   true,
		HttpOnly: true,
	})
	consentUrl := h.oauth2Config.AuthCodeURL(oauthState, oauth2.SetAuthURLParam("prompt", "select_account"))
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (h HTTP) Callback(w http.ResponseWriter, r *http.Request) {
	loginPage := "/"

	redirectURI, err := r.Cookie(RedirectURICookie)
	if err == nil {
		loginPage = loginPage + strings.TrimPrefix(redirectURI.Value, "/")
	}

	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000" + loginPage
	}

	deleteCookie(w, RedirectURICookie, r.Host)
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	oauthCookie, err := r.Cookie(OAuthStateCookie)
	if err != nil {
		h.log.Errorf("Missing oauth state cookie: %v", err)
		http.Redirect(w, r, loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	deleteCookie(w, OAuthStateCookie, r.Host)

	state := r.URL.Query().Get("state")
	if state != oauthCookie.Value {
		h.log.Info("Incoming state does not match local state")
		http.Redirect(w, r, loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	tokens, err := h.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		h.log.Errorf("Exchanging authorization code for tokens: %v", err)
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		h.log.Info("Missing id_token")
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	// Parse and verify ID Token payload.
	idToken, err := h.oauth2Config.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.Info("Invalid id_token")
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	session := &models.Session{
		Token:   generateSecureToken(tokenLength),
		Expires: time.Now().Add(sessionLength),
	}
	if err := idToken.Claims(session); err != nil {
		h.log.WithError(err).Info("Unable to parse claims")
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
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

func deleteCookie(w http.ResponseWriter, name, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
	})
}

func (h HTTP) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "nada_session",
		Value:    "",
		Path:     "/",
		Domain:   r.Host,
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
	})

	session, err := r.Cookie("nada_session")
	if err != nil {
		h.log.WithError(err).Info("Unable to logout session")
	} else if err := h.repo.DeleteSession(r.Context(), session.Value); err != nil {
		h.log.WithError(err).Info("Unable to delete session from database")
	}

	var loginPage string
	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = "/"
	}

	http.Redirect(w, r, loginPage, http.StatusFound)
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
