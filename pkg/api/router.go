package api

import (
	"encoding/json"
	"fmt"
	"io"
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
		r.Use(authMW)
		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getDataproduct(r.Context(), chi.URLParam(r, "id"))
		}))
		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			newDataproduct := NewDataproduct{}
			if err = json.Unmarshal(bodyBytes, &newDataproduct); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}

			return createDataproduct(r.Context(), newDataproduct)
		}))

		r.Delete("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return deleteDataproduct(r.Context(), chi.URLParam(r, "id"))
		}))

		r.Put("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			dp := UpdateDataproduct{}
			if err = json.Unmarshal(bodyBytes, &dp); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}

			return updateDataproduct(r.Context(), chi.URLParam(r, "id"), dp)
		}))

	})

	router.Route("/api/datasets", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getDatasetsMinimal(r.Context())
		}))
		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getDataset(r.Context(), chi.URLParam(r, "id"))
		}))
		r.Post("/{id}/map", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			services := DatasetMap{}
			if err = json.Unmarshal(bodyBytes, &services); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return mapDataset(r.Context(), chi.URLParam(r, "id"), services.Services)
		}))
		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			datasetInput := NewDataset{}
			if err = json.Unmarshal(bodyBytes, &datasetInput); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return createDataset(r.Context(), datasetInput)
		}))
		r.Put("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			datasetInput := UpdateDataset{}
			if err = json.Unmarshal(bodyBytes, &datasetInput); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return updateDataset(r.Context(), chi.URLParam(r, "id"), datasetInput)
		}))
		r.Delete("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return deleteDataset(r.Context(), chi.URLParam(r, "id"))
		}))
		r.Get("/pseudo/accessible", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getAccessiblePseudoDatasetsForUser(r.Context())
		}))
	})

	router.Route("/api/accessRequests", func(r chi.Router) {
		r.Use(authMW)

		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			datasetID := r.URL.Query().Get("datasetId")
			return getAccessRequests(r.Context(), datasetID)
		}))

		r.Post("/process/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			accessRequestID := chi.URLParam(r, "id")
			reason := r.URL.Query().Get("reason")
			action := r.URL.Query().Get("action")
			switch action {
			case "approve":
				return "", approveAccessRequest(r.Context(), accessRequestID)
			case "deny":
				return "", denyAccessRequest(r.Context(), accessRequestID, &reason)
			default:
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("invalid action: %s", action), "Invalid action")
			}
		}))

		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			newAccessRequest := NewAccessRequestDTO{}
			if err = json.Unmarshal(bodyBytes, &newAccessRequest); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return nil, createAccessRequest(r.Context(), newAccessRequest)
		}))

		r.Delete("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			accessRequestID := chi.URLParam(r, "id")
			return nil, deleteAccessRequest(r.Context(), accessRequestID)
		}))

		r.Put("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			updateAccessRequestDTO := UpdateAccessRequestDTO{}
			if err = json.Unmarshal(bodyBytes, &updateAccessRequestDTO); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return nil, updateAccessRequest(r.Context(), updateAccessRequestDTO)
		}))
	})

	router.Route("/api/polly", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			q := r.URL.Query().Get("query")
			return searchPolly(r.Context(), q)
		}))
	})

	router.Route("/api/productareas", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetProductAreas(r.Context())
		}))

		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return GetProductAreaWithAssets(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	router.Route("/api/teamkatalogen", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return SearchTeamKatalogen(r.Context(), r.URL.Query()["gcpGroups"])
		}))
	})

	router.Route("/api/keywords", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getKeywordsListSortedByPopularity(r.Context())
		}))

		r.Post("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			updateKeywordsDto := UpdateKeywordsDto{}
			if err = json.Unmarshal(bodyBytes, &updateKeywordsDto); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}

			return nil, UpdateKeywords(r.Context(), updateKeywordsDto)
		}))
	})

	router.Route("/api/bigquery/columns", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			projectID := r.URL.Query().Get("projectId")
			datasetID := r.URL.Query().Get("datasetId")
			tableID := r.URL.Query().Get("tableId")
			return getBQColumns(r.Context(), projectID, datasetID, tableID)
		}))
	})

	router.Route("/api/bigquery/tables", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			projectID := r.URL.Query().Get("projectId")
			datasetID := r.URL.Query().Get("datasetId")
			return getBQTables(r.Context(), projectID, datasetID)
		}))
	})

	router.Route("/api/bigquery/datasets", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			projectID := r.URL.Query().Get("projectId")
			return getBQDatasets(r.Context(), projectID)
		}))
	})

	router.Route("/api/search", func(r chi.Router) {
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
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
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getUserData(r.Context())
		}))
	})

	router.Route("/api/stories", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getStoryMetadata(r.Context(), chi.URLParam(r, "id"))
		}))
		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			newStory, files, apiErr := parseStoryFilesForm(r.Context(), r)
			if apiErr != nil {
				return nil, apiErr
			}

			return createStory(r.Context(), newStory, files)
		}))
		r.Put("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, err, "Error reading request body")
			}

			input := UpdateStoryDto{}
			if err = json.Unmarshal(bodyBytes, &input); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, err, "Error unmarshalling request body")
			}

			return updateStory(r.Context(), chi.URLParam(r, "id"), input)
		}))
		r.Delete("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return deleteStory(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	router.Route("/api/accesses", func(r chi.Router) {
		r.Use(authMW)
		r.Post("/grant", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			grantAccessData := GrantAccessData{}
			if err = json.Unmarshal(bodyBytes, &grantAccessData); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return nil, grantAccessToDataset(r.Context(), grantAccessData)
		}))

		r.Post("/revoke", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			accessID := r.URL.Query().Get("id")
			return nil, revokeAccessToDataset(r.Context(), accessID)
		}))

	})

	router.Route("/api/pseudo/joinable", func(r chi.Router) {
		r.Use(authMW)
		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			newJoinableView := NewJoinableViews{}
			if err = json.Unmarshal(bodyBytes, &newJoinableView); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}
			return createJoinableViews(r.Context(), newJoinableView)
		}))
		r.Get("/", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getJoinableViewsForUser(r.Context())
		}))
		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getJoinableView(r.Context(), chi.URLParam(r, "id"))
		}))

	})
	router.Route("/api/insightProducts", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return getInsightProduct(r.Context(), chi.URLParam(r, "id"))
		}))

		r.Post("/new", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			input := NewInsightProduct{}
			if err = json.Unmarshal(bodyBytes, &input); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}

			return createInsightProduct(r.Context(), input)
		}))

		r.Put("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
			}

			input := UpdateInsightProductDto{}
			if err = json.Unmarshal(bodyBytes, &input); err != nil {
				return nil, NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
			}

			return updateInsightProduct(r.Context(), chi.URLParam(r, "id"), input)
		}))

		r.Delete("/{id}", apiWrapper(func(r *http.Request) (interface{}, *APIError) {
			return deleteInsightProduct(r.Context(), chi.URLParam(r, "id"))
		}))
	})

	return router
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
