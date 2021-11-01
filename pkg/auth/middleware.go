package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/metadata"
	"github.com/sirupsen/logrus"
)

type contextKey int

const contextUserKey contextKey = 1

type User struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Groups      metadata.Groups
	AccessToken string    `json:"-"`
	Expiry      time.Time `json:"expiry"`
}

func GetUser(ctx context.Context) *User {
	user := ctx.Value(contextUserKey)
	if user == nil {
		return nil
	}
	return user.(*User)
}

type GroupsLister interface {
	GroupsForUser(ctx context.Context, email string) (groups metadata.Groups, err error)
}

type groupsCacheValue struct {
	Groups  metadata.Groups
	Expires time.Time
}

type groupsCacher struct {
	lock  sync.RWMutex
	cache map[string]groupsCacheValue
}

func (g *groupsCacher) Get(email string) ([]metadata.Group, bool) {
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

func (g *groupsCacher) Set(email string, groups []metadata.Group) {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.cache[email] = groupsCacheValue{
		Groups:  groups,
		Expires: time.Now().Add(1 * time.Hour),
	}
}

type Middleware struct {
	tokenVerifier *oidc.IDTokenVerifier
	groupsCache   *groupsCacher
	groupsLister  GroupsLister
}

func newMiddleware(tokenVerifier *oidc.IDTokenVerifier, groupsLister GroupsLister) *Middleware {
	return &Middleware{
		tokenVerifier: tokenVerifier,
		groupsLister:  groupsLister,
		groupsCache: &groupsCacher{
			cache: map[string]groupsCacheValue{},
		},
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
		r = r.WithContext(context.WithValue(ctx, contextUserKey, user))
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) validateUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	// Parse and verify ID Token payload.
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return nil, fmt.Errorf("Unauthorized")
	}

	parts := strings.SplitN(cookie.Value, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("Unauthorized")
	}
	accessToken := parts[0]

	idToken, err := m.tokenVerifier.Verify(r.Context(), parts[1])
	if err != nil {
		return nil, fmt.Errorf("Unauthorized")
	}

	user := &User{
		AccessToken: accessToken,
		Expiry:      idToken.Expiry,
	}
	if err := idToken.Claims(user); err != nil {
		return nil, fmt.Errorf("unable to decode claims: %w", err)
	}

	return user, nil
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
