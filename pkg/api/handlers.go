package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	. "reflect"
	"runtime"
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

var routerMap = map[string]Handler{
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

	"GET /api/accessRequests?{datasetID}": {
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
}

func InstallHanlers(router *chi.Mux) {
	installedRoutes := make(map[string]bool)
	for endpoint, _ := range routerMap {

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
					func(r *http.Request) (interface{}, *service.APIError) {
						return handlerDelegate(handler, pathvars, queryparams, r)
					}))
				log.Info(fmt.Sprintf("Installed path: %v %v, with %v, path vars %v, query params %v", method, path, handlerFuncName(handler), pathvars, queryparams))
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
		for _, queryParam := range queryParams {
			queryParamValue := query.Get(queryParam)
			callParams = append(callParams, ValueOf(queryParamValue))
		}
	}

	if handler.typeDTOIn != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, service.NewAPIError(http.StatusBadRequest, fmt.Errorf("error reading body"), "Error reading request body")
		}

		dto := reflect.New(handler.typeDTOIn).Interface()
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
	return runtime.FuncForPC(reflect.ValueOf(handler.fptr).Pointer()).Name()
}
