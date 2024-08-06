package auth

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type MiddlewareHandler func(http.Handler) http.Handler

type contextKey int

const ContextUserKey contextKey = 1

type CertificateList []*x509.Certificate

type KeyDiscovery struct {
	Keys []Key `json:"keys"`
}

type EncodedCertificate string

type Key struct {
	Kid string               `json:"kid"`
	X5c []EncodedCertificate `json:"x5c"`
}

func FetchCertificates(discoveryURL string, log zerolog.Logger) (map[string]CertificateList, error) {
	log.Info().Msgf("Discover Microsoft signing certificates from %s", discoveryURL)
	azureKeyDiscovery, err := DiscoverURL(discoveryURL)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Decoding certificates for %d keys", len(azureKeyDiscovery.Keys))
	azureCertificates, err := azureKeyDiscovery.Map()
	if err != nil {
		return nil, err
	}
	return azureCertificates, nil
}

// Map transform a KeyDiscovery object into a dictionary with "kid" as key
// and lists of decoded X509 certificates as values.
//
// Returns an error if any certificate does not decode.
func (k *KeyDiscovery) Map() (result map[string]CertificateList, err error) {
	result = make(map[string]CertificateList)

	for _, key := range k.Keys {
		certList := make(CertificateList, 0)
		for _, encodedCertificate := range key.X5c {
			certificate, err := encodedCertificate.Decode()
			if err != nil {
				return nil, err
			}
			certList = append(certList, certificate)
		}
		result[key.Kid] = certList
	}

	return
}

// Decode a base64 encoded certificate into a X509 structure.
func (c EncodedCertificate) Decode() (*x509.Certificate, error) {
	stream := strings.NewReader(string(c))
	decoder := base64.NewDecoder(base64.StdEncoding, stream)
	key, err := io.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(key)
}

func DiscoverURL(url string) (*KeyDiscovery, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return Discover(response.Body)
}

func Discover(reader io.Reader) (*KeyDiscovery, error) {
	document, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	keyDiscovery := &KeyDiscovery{}
	err = json.Unmarshal(document, keyDiscovery)

	return keyDiscovery, err
}

func GetUser(ctx context.Context) *service.User {
	user := ctx.Value(ContextUserKey)
	if user == nil {
		return nil
	}

	return user.(*service.User)
}

func SetUser(ctx context.Context, user *service.User) context.Context {
	return context.WithValue(ctx, ContextUserKey, user)
}

type SessionRetriever interface {
	GetSession(ctx context.Context, token string) (*Session, error)
}

type Middleware struct {
	keyDiscoveryURL string
	tokenVerifier   *oidc.IDTokenVerifier
	groupsCache     *groupsCacher
	azureGroups     *AzureGroupClient
	googleGroups    *GoogleGroupClient
	queries         *gensql.Queries
	log             zerolog.Logger
}

func newMiddleware(
	keyDiscoveryURL string,
	tokenVerifier *oidc.IDTokenVerifier,
	azureGroups *AzureGroupClient,
	googleGroups *GoogleGroupClient,
	querier *gensql.Queries,
	log zerolog.Logger,
) *Middleware {
	return &Middleware{
		keyDiscoveryURL: keyDiscoveryURL,
		tokenVerifier:   tokenVerifier,
		azureGroups:     azureGroups,
		googleGroups:    googleGroups,
		groupsCache: &groupsCacher{
			cache: map[string]groupsCacheValue{},
		},
		queries: querier,
		log:     log,
	}
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return m.handle(next)
}

func (m *Middleware) handle(next http.Handler) http.Handler {
	certificates, err := FetchCertificates(m.keyDiscoveryURL, m.log)
	if err != nil {
		m.log.Fatal().Err(err).Msg("fetching signing certificates from IDP")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token, err := r.Cookie("nada_session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		sess, err := GetSession(ctx, token.Value)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error": "Unable to retrieve session."}`, http.StatusInternalServerError)
			return
		}
		if sess != nil {
			user, err := m.validateUser(certificates, w, sess.AccessToken)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if err := m.addGroupsToUser(ctx, sess.AccessToken, user); err != nil {
				m.log.Error().Err(err).Msg("Unable to add groups")
				w.Header().Add("Content-Type", "application/json")
				http.Error(w, `{"error": "Unable fetch users groups."}`, http.StatusInternalServerError)
				return
			}

			r = r.WithContext(context.WithValue(ctx, ContextUserKey, user))
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) validateUser(certificates map[string]CertificateList, _ http.ResponseWriter, token string) (*service.User, error) {
	var claims jwt.MapClaims

	jwtValidator := JWTValidator(certificates, m.azureGroups.OAuthClientID)

	_, err := jwt.ParseWithClaims(token, &claims, jwtValidator)
	if err != nil {
		return nil, err
	}

	return &service.User{
		Name:   claims["name"].(string),
		Email:  strings.ToLower(claims["preferred_username"].(string)),
		Expiry: time.Unix(int64(claims["exp"].(float64)), 0),
	}, nil
}

func JWTValidator(certificates map[string]CertificateList, audience string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		var certificateList CertificateList
		var kid string
		var ok bool

		if claims, ok := token.Claims.(*jwt.MapClaims); !ok {
			return nil, fmt.Errorf("unable to retrieve claims from token")
		} else {
			if valid := claims.VerifyAudience(audience, true); !valid {
				return nil, fmt.Errorf("the token is not valid for this application")
			}
		}

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		if kid, ok = token.Header["kid"].(string); !ok {
			return nil, fmt.Errorf("field 'kid' is of invalid type %T, should be string", token.Header["kid"])
		}

		if certificateList, ok = certificates[kid]; !ok {
			return nil, fmt.Errorf("kid '%s' not found in certificate list", kid)
		}

		for _, certificate := range certificateList {
			return certificate.PublicKey, nil
		}

		return nil, fmt.Errorf("no certificate candidates for kid '%s'", kid)
	}
}

func (m *Middleware) addGroupsToUser(ctx context.Context, token string, u *service.User) error {
	err := m.addAzureGroups(ctx, token, u)
	if err != nil {
		return fmt.Errorf("unable to add azure groups: %w", err)
	}

	err = m.addGoogleGroups(ctx, u)
	if err != nil {
		return fmt.Errorf("unable to add google groups: %w", err)
	}

	return nil
}

func (m *Middleware) addAzureGroups(ctx context.Context, token string, u *service.User) error {
	groups, ok := m.groupsCache.GetAzureGroups(u.Email)
	if ok {
		u.AzureGroups = groups
		return nil
	}

	groups, err := m.azureGroups.GroupsForUser(ctx, token, u.Email)
	if err != nil {
		return fmt.Errorf("getting groups for user: %w", err)
	}

	m.groupsCache.SetAzureGroups(u.Email, groups)
	u.AzureGroups = groups
	return nil
}

func (m *Middleware) addGoogleGroups(ctx context.Context, u *service.User) error {
	groups, ok := m.groupsCache.GetGoogleGroups(u.Email)
	if !ok {
		var err error
		groups, err = m.googleGroups.Groups(ctx, &u.Email)
		if err != nil {
			return fmt.Errorf("getting groups for user: %w", err)
		}

		m.groupsCache.SetGoogleGroups(u.Email, groups)
	}
	u.GoogleGroups = groups

	allGroups, ok := m.groupsCache.GetGoogleGroups("all")
	if !ok {
		var err error
		allGroups, err = m.googleGroups.Groups(ctx, nil)
		if err != nil {
			return fmt.Errorf("getting all groups: %w", err)
		}

		m.groupsCache.SetGoogleGroups("all", allGroups)
	}
	u.AllGoogleGroups = allGroups

	return nil
}
