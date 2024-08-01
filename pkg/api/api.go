package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/rs/zerolog"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"golang.org/x/oauth2"
)

const (
	tokenLength   = 32
	sessionLength = 7 * time.Hour
)

type OAuth2 interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

type HTTP struct {
	oauth2Config OAuth2
	callbackURL  string
	loginPage    string
	cookies      config.Cookies
	log          zerolog.Logger
}

func NewHTTP(oauth2Config OAuth2, callbackURL string, loginPage string, cookies config.Cookies, log zerolog.Logger) HTTP {
	return HTTP{
		oauth2Config: oauth2Config,
		callbackURL:  callbackURL,
		loginPage:    loginPage,
		cookies:      cookies,
		log:          log,
	}
}

func (h HTTP) deleteCookie(w http.ResponseWriter, name, path, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   domain,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		HttpOnly: true,
	})
}

func (h HTTP) Logout(w http.ResponseWriter, r *http.Request) {
	h.deleteCookie(w, h.cookies.Session.Name, h.cookies.Session.Path, h.cookies.Session.Domain)
	http.Redirect(w, r, h.loginPage, http.StatusFound)
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (h HTTP) Login(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookies.Redirect.Name,
		Value:    r.URL.Query().Get("redirect_uri"),
		Path:     h.cookies.Redirect.Path,
		Domain:   h.cookies.Redirect.Domain,
		MaxAge:   h.cookies.Redirect.MaxAge,
		SameSite: h.cookies.Redirect.GetSameSite(),
		Secure:   h.cookies.Redirect.Secure,
		HttpOnly: h.cookies.Redirect.HttpOnly,
	})

	oauthState := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookies.OauthState.Name,
		Value:    oauthState,
		Path:     h.cookies.OauthState.Path,
		Domain:   h.cookies.OauthState.Domain,
		MaxAge:   h.cookies.OauthState.MaxAge,
		SameSite: h.cookies.OauthState.GetSameSite(),
		Secure:   h.cookies.OauthState.Secure,
		HttpOnly: h.cookies.OauthState.HttpOnly,
	})

	consentUrl := h.oauth2Config.AuthCodeURL(oauthState)
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (h HTTP) Callback(w http.ResponseWriter, r *http.Request) {
	h.deleteCookie(w, h.cookies.Redirect.Name, h.cookies.Redirect.Path, h.cookies.Redirect.Domain)

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	oauthCookie, err := r.Cookie(h.cookies.OauthState.Name)
	if err != nil {
		h.log.Error().Err(err).Msgf("missing oauth state cookie")
		http.Redirect(w, r, h.loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	h.deleteCookie(w, h.cookies.OauthState.Name, h.cookies.OauthState.Path, h.cookies.OauthState.Domain)

	state := r.URL.Query().Get("state")
	if state != oauthCookie.Value {
		h.log.Info().Msg("incoming state does not match local state")
		http.Redirect(w, r, h.loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	tokens, err := h.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		h.log.Error().Err(err).Msg("exchanging authorization code for tokens")
		message := "Internal error: oauth2"
		if strings.HasPrefix(r.Host, "localhost") {
			message = "oauth2 error, try:\n$gcloud auth login --update-adc\n$make env\nbefore running backend"
		}
		http.Error(w, message, http.StatusForbidden)
		return
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		h.log.Info().Msg("missing id_token")
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	// Parse and verify ID Token payload.
	_, err = h.oauth2Config.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.Error().Err(err).Msg("invalid id_token")
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	session := &auth.Session{
		Token:       generateSecureToken(tokenLength),
		Expires:     time.Now().Add(sessionLength),
		AccessToken: tokens.AccessToken,
	}

	b, err := base64.RawStdEncoding.DecodeString(strings.Split(tokens.AccessToken, ".")[1])
	if err != nil {
		h.log.Error().Err(err).Msg("decoding access token")
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	if err := json.Unmarshal(b, session); err != nil {
		h.log.Error().Err(err).Msg("unmarshalling token")
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	if err := auth.CreateSession(r.Context(), session); err != nil {
		h.log.Error().Err(err).Msg("creating session")
		http.Redirect(w, r, h.loginPage+"?error=unauthenticated", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookies.Session.Name,
		Value:    session.Token,
		Path:     h.cookies.Session.Path,
		Domain:   h.cookies.Session.Domain,
		MaxAge:   h.cookies.Session.MaxAge,
		SameSite: h.cookies.Session.GetSameSite(),
		Secure:   h.cookies.Session.Secure,
		HttpOnly: h.cookies.Session.HttpOnly,
	})

	http.Redirect(w, r, h.loginPage, http.StatusFound)
}
