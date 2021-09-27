package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/datakatalogen/backend/auth"
	"github.com/navikt/datakatalogen/backend/config"
	"github.com/navikt/datakatalogen/backend/firestore"
	"github.com/navikt/datakatalogen/backend/middleware"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/go-playground/validator.v9"
)

type authorizer interface {
	UpdateDatastoreAccess(ctx context.Context, datastore map[string]string, accessMap map[string]time.Time) error
	RemoveDatastoreAccess(ctx context.Context, datastore map[string]string, subject string) error
}

type api struct {
	firestore    *firestore.Firestore
	iam          authorizer
	validate     *validator.Validate
	config       config.Config
	teamUUIDs    map[string]string
	teamProjects map[string][]string
}

func ServeStatic(router *chi.Mux) {
	root := "./public"
	fs := http.FileServer(http.Dir(root))

	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(root + r.RequestURI); os.IsNotExist(err) {
			http.StripPrefix(r.RequestURI, fs).ServeHTTP(w, r)
		} else {
			fs.ServeHTTP(w, r)
		}
	})
}

func New(firestore *firestore.Firestore, iam authorizer, config config.Config, teamUUIDs map[string]string, teamProjects map[string][]string) chi.Router {
	api := api{
		firestore:    firestore,
		iam:          iam,
		validate:     validator.New(),
		config:       config,
		teamUUIDs:    teamUUIDs,
		teamProjects: teamProjects,
	}

	azureGroups := auth.AzureGroups{
		Cache:  make(map[string]auth.CacheEntry),
		Client: http.DefaultClient,
		Config: config,
	}

	latencyHistBuckets := []float64{.001, .005, .01, .025, .05, .1, .5, 1, 3, 5}
	prometheusMiddleware := middleware.PrometheusMiddleware("backend", latencyHistBuckets...)
	prometheusMiddleware.Initialize("/api/v1/", http.MethodGet, http.StatusOK)
	authenticatorMiddleware := middleware.JWTValidatorMiddleware(auth.KeyDiscoveryURL(config.OAuth2.TenantID), config.OAuth2.ClientID, config.DevMode, azureGroups, teamUUIDs)

	r := chi.NewRouter()

	r.Use(prometheusMiddleware.Handler())
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	}))

	r.Get("/oauth2/callback", api.callback)
	r.Get("/login", api.login)

	r.Route("/api/v1", func(r chi.Router) {
		// requires valid access token
		r.Group(func(r chi.Router) {
			r.Use(authenticatorMiddleware)
			r.Get("/userinfo", api.userInfo)
		})
		r.Route("/dataproducts", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(authenticatorMiddleware)
				r.Post("/", api.createDataproduct)
				r.Put("/{productID}", api.updateDataproduct)
				r.Delete("/{productID}", api.deleteDataproduct)
			})
			r.Get("/", api.dataproducts)
			r.Get("/{productID}", api.getDataproduct)
		})
		r.Route("/access", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(authenticatorMiddleware)
				r.Delete("/{productID}", api.removeProductAccess)
				r.Post("/{productID}", api.grantProductAccess)
			})

			r.Get("/{productID}", api.getAccessUpdatesForProduct)
		})
	})
	ServeStatic(r)

	return r
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	cfg := auth.CreateOAuth2Config(a.config)
	consentUrl := cfg.AuthCodeURL(a.config.State, oauth2.SetAuthURLParam("redirect_uri", cfg.RedirectURL))
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (a *api) userInfo(w http.ResponseWriter, r *http.Request) {
	var userInfo struct {
		Email       string   `json:"email"`
		Teams       []string `json:"teams"`
		TokenExpiry int      `json:"token_expires"`
	}

	userInfo.Teams = r.Context().Value("teams").([]string)
	userInfo.Email = r.Context().Value("preferred_username").(string)
	userInfo.TokenExpiry = r.Context().Value("token_expiry").(int)

	if err := json.NewEncoder(w).Encode(&userInfo); err != nil {
		log.Errorf("Serializing teams response: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to serialize teams for user\n")
		return
	}
}

func (a *api) callback(w http.ResponseWriter, r *http.Request) {
	cfg := auth.CreateOAuth2Config(a.config)

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		respondf(w, http.StatusForbidden, "No code in query params")
		return
	}

	state := r.URL.Query().Get("state")
	if state != a.config.State {
		log.Errorf("Incoming state does not match local state")
		respondf(w, http.StatusForbidden, "uh oh")
		return
	}

	tokens, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		log.Errorf("Exchanging authorization code for tokens: %v", err)
		respondf(w, http.StatusForbidden, "uh oh")
		return
	}

	domain := a.config.Hostname
	if strings.Contains(domain, "dev.intern.nav.no") {
		domain = "dev.intern.nav.no"
	} else if strings.Contains(domain, "intern.nav.no") {
		domain = "intern.nav.no"
	}

	w.Header().Set("Set-Cookie", fmt.Sprintf("jwt=%v;HttpOnly;Secure;Max-Age=86400;Path=/;Domain=%v", tokens.AccessToken, domain))

	var loginPage string
	if a.config.Hostname == "localhost" {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = fmt.Sprintf("https://%v", a.config.Hostname) // should point to frontend url and not ourselves
	}

	http.Redirect(w, r, loginPage, http.StatusFound) // redirect and set cookie doesn't work on chrome lol
}

func (a *api) requesterHasAccessToDatastore(requestContext context.Context, datastore map[string]string) bool {
	requesterTeams := requestContext.Value("teams").([]string)

	var requesterProjects []string
	for _, team := range requesterTeams {
		requesterProjects = append(requesterProjects, a.teamProjects[team]...)
	}

	return contains(requesterProjects, datastore["project_id"])
}

func (a *api) teamOwnsDatastoreProject(dataproduct firestore.Dataproduct) bool {
	ownerProjectsIDs := a.teamProjects[dataproduct.Team]
	datastoreProjectID := dataproduct.Datastore[0]["project_id"]
	return contains(ownerProjectsIDs, datastoreProjectID)
}

func respondf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	w.WriteHeader(statusCode)

	if _, wErr := w.Write([]byte(fmt.Sprintf(format, args...))); wErr != nil {
		log.Errorf("unable to write response: %v", wErr)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
