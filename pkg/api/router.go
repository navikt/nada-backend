package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/story"
	"github.com/navikt/nada-backend/pkg/teamkatalogen"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type HTTPAPI interface {
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

var ErrUnauthorized = fmt.Errorf("unauthorized")

func New(
	repo *database.Repo,
	gcsClient *gcs.Client,
	teamCatalog teamkatalogen.Teamkatalogen,
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	gqlServer *handler.Server,
	promReg *prometheus.Registry,
	amplitudeClient amplitude.Amplitude,
	teamTokenCreds string,
	log *logrus.Logger,
) *chi.Mux {
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	storyHandler := story.NewHandler(repo, gcsClient, teamCatalog, amplitudeClient, log.WithField("subsystem", "story"))

	router := chi.NewRouter()
	router.Use(corsMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.Handle("/query", authMW(gqlServer))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
	})
	router.Route(`/{story|quarto}/`, func(r chi.Router) {
		r.Use(storyHandler.Middleware)
		r.Get("/*", storyHandler.GetObject)
		r.Post("/create", storyHandler.Create)
		r.Route("/update/{id}", func(r chi.Router) {
			r.Put("/", storyHandler.Update)
			r.Patch("/", storyHandler.Append)
		})
	})
	router.Route("/internal", func(r chi.Router) {
		r.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
		r.Get("/teamtokens", func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			authHeaderParts := strings.Split(authHeader, " ")
			if len(authHeaderParts) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if authHeaderParts[1] != teamTokenCreds {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenTeamMap, err := repo.GetNadaTokens(r.Context())
			if err != nil {
				log.WithError(err).Error("getting nada tokens")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			payloadBytes, err := json.Marshal(tokenTeamMap)
			if err != nil {
				log.WithError(err).Error("marshalling nada token map reponse")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payloadBytes)
		})
	})

	router.Route("/api/dataproducts", func(r chi.Router) {
		r.Get("/{id}", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetDataproduct(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	router.Route("/api/datasets", func(r chi.Router) {
		r.Get("/{id}", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetDataset(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	router.Route("/api/productareas", func(r chi.Router) {
		r.Get("/", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetProductAreas(r.Context())
		}))

		r.Get("/{id}", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetProductAreaWithAssets(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	router.Route("/api/teamkatalogen", func(r chi.Router) {
		r.Get("/", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return SearchTeamKatalogen(r.Context(), r.URL.Query()["gcpGroups"])
		}))
	})

	router.Route("/api/keywords", func(r chi.Router) {
		r.Get("/", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getKeywordsListSortedByPopularity(r.Context())
		}))
	})

	router.Route("/api/search", func(r chi.Router) {
		r.Get("/", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			searchOptions, err := parseSearchOptionsFromRequest(r)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, err, "Failed to parse search options")
			}

			return Search(r.Context(), searchOptions)
		}))
	})

	router.Route("/api/user", func(r chi.Router) {
		r.Use(authMW)
		r.Put("/token", func(w http.ResponseWriter, r *http.Request) {
			team := r.URL.Query().Get("team")
			if apiErr := RotateNadaToken(r.Context(), team); apiErr != nil {
				http.Error(w, apiErr.Error(), apiErr.HttpStatus)
				return
			}
		})
	})

	router.Route("/api/userData", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", apiGetWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getUserData(r.Context())
		}))
	})

	return router
}

func ensureUserInGroup(ctx context.Context, group string) error {
	user := auth.GetUser(ctx)
	if user == nil || !user.GoogleGroups.Contains(group) {
		return ErrUnauthorized
	}
	return nil
}

func parseSearchOptionsFromRequest(r *http.Request) (*SearchOptions, error) {
	query := r.URL.Query()

	options := SearchOptions{}

	// Parse 'text' parameter
	if text, ok := query["text"]; ok && len(text) > 0 {
		options.Text = text[0]
	}

	// Parse 'keywords' parameter
	if keywords, ok := query["keywords"]; ok && len(keywords) > 0 {
		options.Keywords = strings.Split(keywords[0], ",")
	}

	// Parse 'groups' parameter
	if groups, ok := query["groups"]; ok && len(groups) > 0 {
		options.Groups = strings.Split(groups[0], ",")
	}

	// Parse 'teamIDs' parameter
	if teamIDs, ok := query["teamIDs"]; ok && len(teamIDs) > 0 {
		options.TeamIDs = strings.Split(teamIDs[0], ",")
	}

	// Parse 'services' parameter
	if services, ok := query["services"]; ok && len(services) > 0 {
		options.Services = strings.Split(services[0], ",")
	}

	// Parse 'types' parameter
	if types, ok := query["types"]; ok && len(types) > 0 {
		options.Types = strings.Split(types[0], ",")
	}

	// Parse 'limit' parameter
	if limit, ok := query["limit"]; ok && len(limit) > 0 {
		limitVal, err := strconv.Atoi(limit[0])
		if err != nil {
			return nil, err // Handle or return an error appropriately
		}
		options.Limit = &limitVal
	}

	// Parse 'offset' parameter
	if offset, ok := query["offset"]; ok && len(offset) > 0 {
		offsetVal, err := strconv.Atoi(offset[0])
		if err != nil {
			return nil, err // Handle or return an error appropriately
		}
		options.Offset = &offsetVal
	}

	return &options, nil
}
