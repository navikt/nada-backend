package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type OAuth2 interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

type HTTP struct {
	oauth2Config OAuth2
	log          *logrus.Entry
}

func new(oauth2Config OAuth2, log *logrus.Entry) HTTP {
	return HTTP{
		oauth2Config: oauth2Config,
		log:          log,
	}
}

func (h *HTTP) Login(w http.ResponseWriter, r *http.Request) {
	oauthState := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauthstate",
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

func (h *HTTP) Callback(w http.ResponseWriter, r *http.Request) {
	var loginPage string
	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = "/"
	}

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	oauthCookie, err := r.Cookie("oauthstate")
	if err != nil {
		h.log.Errorf("Missing oauth state cookie: %v", err)
		http.Redirect(w, r, loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauthstate",
		Value:    "",
		Path:     "/",
		Domain:   r.Host,
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
	})

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
	_, err = h.oauth2Config.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.Info("Invalid id_token")
		http.Redirect(w, r, loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	// TODO(thokra): Use secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokens.AccessToken + "|" + rawIDToken,
		Path:     "/",
		Domain:   r.Host,
		Expires:  tokens.Expiry,
		Secure:   true,
		HttpOnly: true,
	})

	http.Redirect(w, r, loginPage, http.StatusFound)
}

func (h *HTTP) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
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
