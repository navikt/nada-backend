package api

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
	"golang.org/x/oauth2"
)

type GCP interface {
	GetDataset(ctx context.Context, projectID, datasetID string) ([]openapi.BigqueryTypeMetadata, error)
	GetDatasets(ctx context.Context, projectID string) ([]string, error)
	GetTables(ctx context.Context, projectID string) ([]gensql.DatasourceBigquery, error)
}

type OAuth2 interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

/*import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
)

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	consentUrl := s.oauth2Config.AuthCodeURL("banan")
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (s *Server) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "No code in query params", http.StatusForbidden)
		return
	}

	// TODO(thokra): Introduce varying state
	state := r.URL.Query().Get("state")
	if state != "banan" {
		s.log.Info("Incoming state does not match local state")
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	tokens, err := s.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		s.log.Errorf("Exchanging authorization code for tokens: %v", err)
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		s.log.Info("Missing id_token")
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	// Parse and verify ID Token payload.
	_, err = s.oauth2Config.Verify(r.Context(), rawIDToken)
	if err != nil {
		s.log.Info("Invalid id_token")
		http.Error(w, "uh oh", http.StatusForbidden)
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

	var loginPage string
	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = "/"
	}
	http.Redirect(w, r, loginPage, http.StatusFound)
}

func defaultInt(i *int, def int) int {
	if i != nil {
		return *i
	}
	return def
}
*/
