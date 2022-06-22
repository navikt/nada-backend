package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/jwtauth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
)

type contextKey int

const ContextUserKey contextKey = 1

type User struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	AzureGroups  Groups
	GoogleGroups Groups
}

func GetUser(ctx context.Context) *User {
	user := ctx.Value(ContextUserKey)
	if user == nil {
		return nil
	}
	return user.(*User)
}

type SessionRetriever interface {
	GetSession(ctx context.Context, token string) (*models.Session, error)
}

type Middleware struct {
	tokenVerifier *oidc.IDTokenVerifier
	groupsCache   *groupsCacher
	azureGroups   *AzureGroupClient
	googleGroups  *GoogleGroupClient
	sessionStore  SessionRetriever
}

func newMiddleware(tokenVerifier *oidc.IDTokenVerifier, azureGroups *AzureGroupClient, googleGroups *GoogleGroupClient, sessionStore SessionRetriever) *Middleware {
	return &Middleware{
		tokenVerifier: tokenVerifier,
		azureGroups:   azureGroups,
		googleGroups:  googleGroups,
		groupsCache: &groupsCacher{
			cache: map[string]groupsCacheValue{},
		},
		sessionStore: sessionStore,
	}
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return m.handle(next)
}

func (m *Middleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := jwtauth.TokenFromCookie(r)

		user, err := m.validateUser(w, token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if err := m.addGroupsToUser(ctx, token, user); err != nil {
			logrus.WithError(err).Error("Unable to add groups")
			w.Header().Add("Content-Type", "application/json")
			http.Error(w, `{"error": "Unable fetch users groups."}`, http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(ctx, ContextUserKey, user))
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) validateUser(w http.ResponseWriter, token string) (*User, error) {
	tokenParsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return nil, nil
	})

	email := ""
	name := ""
	if claims, ok := tokenParsed.Claims.(jwt.MapClaims); ok {
		email = strings.ToLower(claims["preferred_username"].(string))
		name = claims["name"].(string)
	} else {
		return nil, err
	}

	return &User{
		Name:  name,
		Email: email,
	}, nil
}

func (m *Middleware) addGroupsToUser(ctx context.Context, token string, u *User) error {
	if err := m.addAzureGroups(ctx, token, u); err != nil {
		return err
	}
	if err := m.addGoogleGroups(ctx, u); err != nil {
		return err
	}

	return nil
}

func (m *Middleware) addAzureGroups(ctx context.Context, token string, u *User) error {
	groups, ok := m.groupsCache.GetAzureGroups(u.Email)
	if ok {
		u.AzureGroups = groups
		return nil
	}

	groups, err := m.azureGroups.GroupsForUser(ctx, token, u.Email)
	if err != nil {
		return err
	}

	m.groupsCache.SetAzureGroups(u.Email, groups)
	u.AzureGroups = groups
	return nil
}

func (m *Middleware) addGoogleGroups(ctx context.Context, u *User) error {
	groups, ok := m.groupsCache.GetGoogleGroups(u.Email)
	if ok {
		u.GoogleGroups = groups
		return nil
	}

	groups, err := m.googleGroups.GroupsForUser(ctx, u.Email)
	if err != nil {
		return err
	}

	m.groupsCache.SetGoogleGroups(u.Email, groups)
	u.GoogleGroups = groups
	return nil
}
