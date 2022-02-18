package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
)

type contextKey int

const ContextUserKey contextKey = 1

type User struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Groups bigquery.Groups
	Expiry time.Time `json:"expiry"`
}

func GetUser(ctx context.Context) *User {
	user := ctx.Value(ContextUserKey)
	if user == nil {
		return nil
	}
	return user.(*User)
}

type GroupsLister interface {
	GroupsForUser(ctx context.Context, email string) (groups bigquery.Groups, err error)
}

type groupsCacheValue struct {
	Groups  bigquery.Groups
	Expires time.Time
}

type groupsCacher struct {
	lock  sync.RWMutex
	cache map[string]groupsCacheValue
}

func (g *groupsCacher) Get(email string) ([]bigquery.Group, bool) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	v, ok := g.cache[email]
	if !ok {
		return nil, false
	}
	if v.Expires.Before(time.Now()) {
		return nil, false
	}
	return v.Groups, true
}

func (g *groupsCacher) Set(email string, groups []bigquery.Group) {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.cache[email] = groupsCacheValue{
		Groups:  groups,
		Expires: time.Now().Add(1 * time.Hour),
	}
}

type SessionRetriever interface {
	GetSession(ctx context.Context, token string) (*models.Session, error)
}

type Middleware struct {
	tokenVerifier *oidc.IDTokenVerifier
	groupsCache   *groupsCacher
	groupsLister  GroupsLister
	sessionStore  SessionRetriever
}

func newMiddleware(tokenVerifier *oidc.IDTokenVerifier, groupsLister GroupsLister, sessionStore SessionRetriever) *Middleware {
	return &Middleware{
		tokenVerifier: tokenVerifier,
		groupsLister:  groupsLister,
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
		user, err := m.validateUser(w, r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if err := m.addGroupsToUser(r.Context(), user); err != nil {
			logrus.WithError(err).Error("Unable to add groups")
			w.Header().Add("Content-Type", "application/json")
			http.Error(w, `{"error": "Unable fetch users groups."}`, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		r = r.WithContext(context.WithValue(ctx, ContextUserKey, user))
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) validateUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	// Parse and verify ID Token payload.
	cookie, err := r.Cookie("nada_session")
	if err != nil {
		return nil, fmt.Errorf("Unauthorized")
	}

	session, err := m.sessionStore.GetSession(r.Context(), cookie.Value)
	if err != nil {
		return nil, err
	}

	return &User{
		Name:   session.Name,
		Email:  session.Email,
		Expiry: session.Expires,
	}, nil
}

func (m *Middleware) addGroupsToUser(ctx context.Context, u *User) error {
	groups, ok := m.groupsCache.Get(u.Email)
	if ok {
		u.Groups = groups
		return nil
	}

	groups, err := m.groupsLister.GroupsForUser(ctx, u.Email)
	if err != nil {
		return err
	}

	m.groupsCache.Set(u.Email, groups)
	u.Groups = groups
	return nil
}
