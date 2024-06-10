package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	. "reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/service"
)

type Handler struct {
	//DTO type of the handler
	typeDTOIn Type

	//Handler function
	fptr any
}

// Map of all the handlers
//
// the keys of the map are the endpoints, which follow the format:
//
//	<METHOD> /<PATH>/<PATH_VAR1>/<PATH_VAR2>?<QUERY_PARAM1>&<QUERY_PARAM2>
//	METHOD is the HTTP method
//	PATH is the path of the endpoint
//	PATH_VAR1, PATH_VAR2 are path variable names, which will be used as as parameters following "context" that is the first parameter of the handler function
//	QUERY_PARAM1, QUERY_PARAM2 are query parameters names, which will be used as as parameters following the path variables
//
// The values of the map are the handlers, which contain the function pointer and the DTO type of the handler
// The function pointer (fptr) is the function that will be called when the endpoint is hit, and the function must have context as its first parameter.
// There can be a list of string parameters, which are the path variables and query parameters, that will be passed to the function.
// The last parameter must be a DTO type which will be posted by the frontend as request body. The parameter must NOT be a pointer to DTO!
// The handler function must return a tuple of a response DTO and an *APIError.
var routerMap = map[string]Handler{
	//dataproducts
	"GET /api/dataproducts/{id}": {
		fptr: service.GetDataproduct,
	},
	"POST /api/dataproducts/new": {
		typeDTOIn: TypeOf(service.NewDataproduct{}),
		fptr:      service.CreateDataproduct,
	},
	"DELETE /api/dataproducts/{id}": {
		fptr: service.DeleteDataproduct,
	},
	"PUT /api/dataproducts/{id}": {
		typeDTOIn: TypeOf(service.UpdateDataproductDto{}),
		fptr:      service.UpdateDataproduct,
	},

	//datasets
	"POST /api/datasets/new": {
		typeDTOIn: TypeOf(service.NewDataset{}),
		fptr:      service.CreateDataset,
	},
	"GET /api/datasets/{id}": {
		fptr: service.GetDataset,
	},
	"GET /api/datasets": {
		fptr: service.GetDatasetsMinimal,
	},
	"POST /api/datasets/{id}/map": {
		typeDTOIn: TypeOf(service.DatasetMap{}),
		fptr:      service.MapDataset,
	},
	"PUT /api/datasets/{id}": {
		typeDTOIn: TypeOf(service.UpdateDatasetDto{}),
		fptr:      service.UpdateDataset,
	},
	"DELETE /api/datasets/{id}": {
		fptr: service.DeleteDataset,
	},
	"GET /api/datasets/pseudo/accessible": {
		fptr: service.GetAccessiblePseudoDatasetsForUser,
	},

	//accessRequests
	"GET /api/accessRequests?{datasetId}": {
		fptr: service.GetAccessRequests,
	},
	"POST /api/accessRequests/{id}/process?{action}&{reason}": {
		fptr: service.ProcessAccessRequest,
	},
	"POST /api/accessRequests/new": {
		typeDTOIn: TypeOf(service.NewAccessRequestDTO{}),
		fptr:      service.CreateAccessRequest,
	},
	"DELETE /api/accessRequests/{id}": {
		fptr: service.DeleteAccessRequest,
	},
	"PUT /api/accessRequests/{id}": {
		typeDTOIn: TypeOf(service.UpdateAccessRequestDTO{}),
		fptr:      service.UpdateAccessRequest,
	},

	//polly
	"GET /api/polly?{query}": {
		fptr: service.SearchPolly,
	},

	//productAreas
	"GET /api/productareas": {
		fptr: service.GetProductAreas,
	},
	"GET /api/productareas/{id}": {
		fptr: service.GetProductAreaWithAssets,
	},

	//teamkatalogen
	"GET /api/teamkatalogen?{[gcpGroups]}": {
		fptr: service.SearchTeamKatalogen,
	},

	//keywords
	"GET /api/keywords": {
		fptr: service.GetKeywordsListSortedByPopularity,
	},
	"POST /api/keywords": {
		typeDTOIn: TypeOf(service.UpdateKeywordsDto{}),
		fptr:      service.UpdateKeywords,
	},

	//bigquery
	"GET /api/bigquery/columns?{projectId}&{datasetId}&{tableId}": {
		fptr: service.GetBQColumns,
	},
	"GET /api/bigquery/tables?{projectId}&{datasetId}": {
		fptr: service.GetBQTables,
	},
	"GET /api/bigquery/datasets?{projectId}": {
		fptr: service.GetBQDatasets,
	},

	//user
	"PUT /api/user/token?{team}": {
		fptr: service.RotateNadaToken,
	},
	"GET /api/userData": {
		fptr: service.GetUserData,
	},

	//slack
	"GET /api/slack/isValid?{channel}": {
		fptr: service.IsValidSlackChannel,
	},

	//stories
	"GET /api/stories/{id}": {
		fptr: service.GetStoryMetadata,
	},
	"POST /api/stories/new": {
		typeDTOIn: TypeOf(service.NewStory{}),
		fptr:      service.CreateStory,
	},
	"DELETE /api/stories/{id}": {
		fptr: service.DeleteStory,
	},

	//accesses
	"POST /api/accesses/grant": {
		typeDTOIn: TypeOf(service.GrantAccessData{}),
		fptr:      service.GrantAccessToDataset,
	},
	"POST /api/accesses/revoke?{id}": {
		fptr: service.RevokeAccessToDataset,
	},

	//pseudo
	"POST /api/pseudo/joinable/new": {
		typeDTOIn: TypeOf(service.NewJoinableViews{}),
		fptr:      service.CreateJoinableViews,
	},
	"GET /api/pseudo/joinable": {
		fptr: service.GetJoinableViewsForUser,
	},
	"GET /api/pseudo/joinable/{id}": {
		fptr: service.GetJoinableView,
	},

	//insightProducts
	"GET /api/insightProducts/{id}": {
		fptr: service.GetInsightProduct,
	},
	"POST /api/insightProducts/new": {
		typeDTOIn: TypeOf(service.NewInsightProduct{}),
		fptr:      service.CreateInsightProduct,
	},
	"PUT /api/insightProducts/{id}": {
		typeDTOIn: TypeOf(service.UpdateInsightProductDto{}),
		fptr:      service.UpdateInsightProduct,
	},
	"DELETE /api/insightProducts/{id}": {
		fptr: service.DeleteInsightProduct,
	},

	//search
	"GET /api/search?{text}&{[keywords]}&{[groups]}&{[teamIDs]}&{[services]}&{[types]}&{(limit)}&{(offset)}": {
		fptr: service.Search,
	},
}

func MountHandlers(router *chi.Mux) {
	installedRoutes := make(map[string]bool)
	for endpoint := range routerMap {

		_, path, _, _, err := parseEndpoint(endpoint)
		if err != nil {
			log.Errorf("Error parsing endpoint: %s", err.Error())
			continue
		}

		if _, ok := installedRoutes[path]; ok {
			continue
		}

		installedRoutes[path] = true

		router.Route(path, func(r chi.Router) {
			for e, handler := range routerMap {
				method, p, pathvars, queryparams, _ := parseEndpoint(e)
				if p != path {
					continue
				}
				var routerFn func(string, http.HandlerFunc)
				switch method {
				case "POST":
					routerFn = r.Post
				case "GET":
					routerFn = r.Get
				case "PUT":
					routerFn = r.Put
				case "DELETE":
					routerFn = r.Delete
				case "PATCH":
					routerFn = r.Patch
				default:
					log.Errorf("Invalid method: %s", method)
					return
				}
				routerFn("/", apiWrapper(
					func(r *http.Request, payload any) (interface{}, *service.APIError) {
						return handlerDelegate(payload.(Handler), pathvars, queryparams, r)
					}, handler))
				log.Info(fmt.Sprintf("Mounted path: %v %v, with %v, path vars %v, query params %v", method, path, handlerFuncName(handler), pathvars, queryparams))
			}
		})
	}
}

// Function to parse the REST call string
func parseEndpoint(endpoint string) (string, string, []string, []string, error) {
	// Split the string by spaces to separate METHOD and PATH
	parts := strings.SplitN(endpoint, " ", 2)
	if len(parts) < 2 {
		return "", "", nil, nil, fmt.Errorf("invalid endpoint: %s", endpoint)
	}

	method := parts[0]
	pathAndQuery := parts[1]

	// Split the path and query parameters
	pathParts := strings.Split(pathAndQuery, "?")
	path := pathParts[0]

	var pathVariables []string
	pathSegments := strings.Split(path, "/")
	for _, segment := range pathSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			pathVariables = append(pathVariables, strings.Trim(segment, "{}"))
		}
	}

	var queryVariables []string
	if len(pathParts) > 1 {
		queryParams := pathParts[1]
		querySegments := strings.Split(queryParams, "&")
		for _, segment := range querySegments {
			queryVariables = append(queryVariables, strings.Trim(segment, "{}"))
		}
	}

	return method, path, pathVariables, queryVariables, nil
}

func buildParamsForFunction(handler Handler, pathVars []string, queryParams []string, r *http.Request) ([]Value, *service.APIError) {
	callParams := []Value{
		ValueOf(r.Context()),
	}
	if len(pathVars) > 0 {
		for _, pathVar := range pathVars {
			pathVarValue := chi.URLParam(r, pathVar)
			callParams = append(callParams, ValueOf(pathVarValue))
		}
	}

	if len(queryParams) > 0 {
		query := r.URL.Query()
		fmt.Println(query)
		for _, queryParam := range queryParams {
			name, isArray := extractArrayName(queryParam)
			fmt.Println(name, isArray)
			var queryParamValue any = ""
			if isArray {
				queryParamValue = []string{}
				if query.Has(name) {
					queryParamValue = strings.Split(query[name][0], ",")
				}
			} else {
				name, isInt := extractIntVariableName(queryParam)
				fmt.Println(name, isInt)
				if isInt {
					queryParamValue = 0
					if query.Has(name) {
						queryParamIntValue, err := strconv.Atoi(query[name][0])
						if err != nil {
							return nil, service.NewAPIError(http.StatusBadRequest, fmt.Errorf("error parsing query parameter"), "Error parsing query parameter")
						}
						queryParamValue = queryParamIntValue
					}
				} else {
					queryParamValue = ""
					fmt.Println(name)
					fmt.Print(query)
					fmt.Println(query.Has(name))
					if query.Has(name) {
						queryParamValue = query[name][0]
						fmt.Println(queryParamValue)
					}
				}
			}
			callParams = append(callParams, ValueOf(queryParamValue))
		}
	}

	if handler.typeDTOIn != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, service.NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
		}

		dto := New(handler.typeDTOIn).Interface()
		if err = json.Unmarshal(bodyBytes, dto); err != nil {
			return nil, service.NewAPIError(http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"), "Error unmarshalling request body")
		}
		callParams = append(callParams, ValueOf(dto).Elem())
	}
	return callParams, nil
}

func buildAndCallFunction(handler Handler, funcParam []Value) (interface{}, *service.APIError) {
	fn := ValueOf(handler.fptr)
	if fn.Kind() != Func {
		return nil, service.NewAPIError(http.StatusInternalServerError, fmt.Errorf("invalid endpoint"), "invalid endpoint")
	}
	fmt.Println("Calling function: ", handlerFuncName(handler))
	re := fn.Call(funcParam)
	outDto := re[0].Interface()
	apiErr := re[1].Interface().(*service.APIError)
	return outDto, apiErr
}

func handlerDelegate(handler Handler, pathvars []string, queryparams []string, r *http.Request) (interface{}, *service.APIError) {
	callParams, err := buildParamsForFunction(handler, pathvars, queryparams, r)
	if err != nil {
		return nil, err
	}
	return buildAndCallFunction(handler, callParams)
}

func handlerFuncName(handler Handler) string {
	return runtime.FuncForPC(ValueOf(handler.fptr).Pointer()).Name()
}

func apiWrapper(handlerDelegate func(*http.Request, any) (interface{}, *service.APIError), handlerParam any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dto, apiErr := handlerDelegate(r, handlerParam)
		if apiErr != nil {
			apiErr.Log()
			http.Error(w, apiErr.Error(), apiErr.HttpStatus)
			return
		}
		if dto != nil {
			err := json.NewEncoder(w).Encode(dto)
			if err != nil {
				log.WithError(err).Error("Failed to encode response")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func extractArrayName(s string) (string, bool) {
	re := regexp.MustCompile(`^\[(.*)\]$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 0 {
		return matches[1], true
	}
	return s, false
}

func extractIntVariableName(s string) (string, bool) {
	re := regexp.MustCompile(`^\((.*)\)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 0 {
		return matches[1], true
	}
	return s, false
}
