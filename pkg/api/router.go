package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/service"
	. "github.com/navikt/nada-backend/pkg/service"
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

func NewRouter(
	repo *database.Repo,
	gcsClient *gcs.Client,
	teamCatalog teamkatalogen.Teamkatalogen,
	httpAPI HTTPAPI,
	authMW auth.MiddlewareHandler,
	promReg *prometheus.Registry,
	amplitudeClient amplitude.Amplitude,
	teamTokenCreds string,
	l *logrus.Logger,
) *chi.Mux {
	log = l
	corsMW := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	router := chi.NewRouter()
	router.Use(corsMW)
	router.Use(authMW)
	router.Route("/api", func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/api/query"))
		r.HandleFunc("/login", httpAPI.Login)
		r.HandleFunc("/oauth2/callback", httpAPI.Callback)
		r.HandleFunc("/logout", httpAPI.Logout)
	})
	MountHandlers(router)
	router.Route(`/{story|quarto}/`, func(r chi.Router) {
		r.Use(StoryHTTPMiddleware)
		r.Get("/*", GetGCSObject)
		r.Post("/create", CreateStoryHTTP)
		r.Route("/update/{id}", func(r chi.Router) {
			r.Put("/", UpdateStoryHTTP)
			r.Patch("/", AppendFileHTTP)
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

			tokenTeamMap, err := service.GetNadaTokens(r.Context())
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

	router.Route("/api/stories", func(r chi.Router) {
		r.Use(authMW)
		r.Post("/new", apiWrapper(func(r *http.Request, handlerParam any) (interface{}, *APIError) {
			newStory, files, apiErr := ParseStoryFilesForm(r.Context(), r)
			if apiErr != nil {
				return nil, apiErr
			}

			return CreateStory(r.Context(), newStory, files)
		}, nil))
	})

	router.Route("/api/bigquery/tables/sync", func(r chi.Router) {
		r.Post("/", apiWrapper(func(r *http.Request, payload any) (interface{}, *APIError) {
			bqs, err := GetBigqueryDatasources(r.Context())
			if err != nil {
				return false, err
			}

			var errs ErrorList

			for _, bq := range bqs {
				err := UpdateMetadata(r.Context(), bq)
				if err != nil {
					errs = HandleSyncError(r.Context(), errs, err, bq)
				}
			}
			if len(errs) != 0 {
				return false, NewAPIError(http.StatusInternalServerError, errs, "Failed to sync bigquery tables")
			}

			return true, nil
		}, nil))
	})
	return router
}
