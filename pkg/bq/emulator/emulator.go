package emulator

import (
	"cloud.google.com/go/iam/apiv1/iampb"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httputil"
	"testing"
)

type emulator struct {
	handler    http.Handler
	testServer *server.TestServer
	emulator   *server.Server
	t          *testing.T
}

type EndpointMock struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

type Dataset struct {
	DatasetID string
	TableID   string
	Columns   []*types.Column
}

func ColumnNullable(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.NullableMode,
	}
}

func ColumnRequired(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.RequiredMode,
	}
}

func ColumnRepeated(name string) *types.Column {
	return &types.Column{
		Name: name,
		Type: types.STRING,
		Mode: types.RepeatedMode,
	}
}

func (e *emulator) EnableMock(debugRequest bool, log zerolog.Logger, mocks ...*EndpointMock) {
	log.Info().Msg("Enabling mocks")

	handler := e.emulator.Handler

	router := chi.NewRouter()

	debugFn := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if debugRequest {
				request, err := httputil.DumpRequest(r, true)
				if err != nil {
					log.Error().Err(err).Msg("Failed to dump request")
				}

				fmt.Println(string(request))
			}

			next.ServeHTTP(w, r)
		})
	}

	for _, mock := range mocks {
		log.Info().Msgf("Adding mock endpoint: %s %s", mock.Method, mock.Path)
		router.With(debugFn).MethodFunc(mock.Method, mock.Path, mock.Handler)
	}

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msgf("No mocked endpoint found, forwarding to emulator: %s", r.URL.Path)

		if debugRequest {
			request, err := httputil.DumpRequest(r, true)
			if err != nil {
				log.Error().Err(err).Msg("Failed to dump request")
			}
			fmt.Println(string(request))
		}

		handler.ServeHTTP(w, r)
	})

	e.emulator.Handler = router
	e.testServer.Close()
	e.testServer = e.emulator.TestServer()
}

func (e *emulator) Cleanup() {
	e.testServer.Close()
}

func (e *emulator) Endpoint() string {
	return e.testServer.URL
}

func (e *emulator) WithProject(projectID string, datasets ...*Dataset) {
	p := &types.Project{
		ID: projectID,
	}

	for _, ds := range datasets {
		if ds == nil {
			continue
		}

		d := &types.Dataset{
			ID: ds.DatasetID,
		}

		if ds.TableID != "" {
			t := &types.Table{
				ID: ds.TableID,
			}

			for _, col := range ds.Columns {
				t.Columns = append(t.Columns, col)
			}

			d.Tables = append(d.Tables, t)
		}

		p.Datasets = append(p.Datasets, d)
	}

	e.WithSource(p.ID, server.StructSource(p))
}

func (e *emulator) WithSource(projectID string, source server.Source) {
	err := e.emulator.Load(source)
	if err != nil {
		e.t.Fatalf("initializing bigquery emulator: %v", err)
	}

	if err := e.emulator.SetProject(projectID); err != nil {
		e.t.Fatalf("setting project: %v", err)
	}

	e.testServer = e.emulator.TestServer()
}

func New(t *testing.T) *emulator {
	s, err := server.New(server.TempStorage)
	if err != nil {
		t.Fatalf("creating bigquery emulator: %v", err)
	}

	return &emulator{
		t:        t,
		emulator: s,
	}
}

func DatasetTableIAMPolicyGetMock(log zerolog.Logger, policy *iampb.Policy) *EndpointMock {
	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(policy)
		if err != nil {
			log.Error().Err(err).Msg("Failed to encode policy")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	return &EndpointMock{
		Method:  http.MethodPost,
		Path:    "/projects/test-project/datasets/test-dataset/tables/test-table:getIamPolicy",
		Handler: handlerFn,
	}
}

func DatasetTableIAMPolicySetMock(log zerolog.Logger, into *iampb.SetIamPolicyRequest) *EndpointMock {
	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(into)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decode policy")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If the header is StatusNoContent, the body is ignored by the google client
		// otherwise it will try to parse the body
		w.WriteHeader(http.StatusNoContent)
	}

	return &EndpointMock{
		Method:  http.MethodPost,
		Path:    "/projects/test-project/datasets/test-dataset/tables/test-table:setIamPolicy",
		Handler: handlerFn,
	}
}
