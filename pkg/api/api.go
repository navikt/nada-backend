package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
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
	callbackURL  string
	log          *logrus.Entry
	repo         *database.Repo
}

func NewHTTP(oauth2Config OAuth2, callbackURL string, repo *database.Repo, log *logrus.Entry) HTTP {
	return HTTP{
		oauth2Config: oauth2Config,
		callbackURL:  callbackURL,
		log:          log,
		repo:         repo,
	}
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
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	deleteCookie(w, "jwt", host)

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

func (h HTTP) Login(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	redirectURI := r.URL.Query().Get("redirect_uri")
	http.SetCookie(w, &http.Cookie{
		Name:     RedirectURICookie,
		Value:    redirectURI,
		Path:     "/",
		Domain:   host,
		Expires:  time.Now().Add(30 * time.Minute),
		Secure:   true,
		HttpOnly: true,
	})

	oauthState := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookie,
		Value:    oauthState,
		Path:     "/",
		Domain:   host,
		Expires:  time.Now().Add(30 * time.Minute),
		Secure:   true,
		HttpOnly: true,
	})

	consentUrl := h.oauth2Config.AuthCodeURL(oauthState, oauth2.SetAuthURLParam("redirect_uri", h.callbackURL))
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (h HTTP) Callback(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	loginPage := "/"

	redirectURI, err := r.Cookie(RedirectURICookie)
	if err == nil {
		loginPage = loginPage + strings.TrimPrefix(redirectURI.Value, "/")
	}

	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000" + loginPage
	}

	deleteCookie(w, RedirectURICookie, host)
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

	deleteCookie(w, OAuthStateCookie, host)

	state := r.URL.Query().Get("state")
	if state != oauthCookie.Value {
		h.log.Info("Incoming state does not match local state")
		http.Redirect(w, r, loginPage+"?error=invalid-state", http.StatusFound)
		return
	}

	tokens, err := h.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		h.log.Errorf("Exchanging authorization code for tokens: %v", err)
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokens.AccessToken,
		Path:     "/",
		Domain:   host,
		MaxAge:   86400,
		Secure:   true,
		HttpOnly: true,
	})

	http.Redirect(w, r, loginPage, http.StatusFound)
}
