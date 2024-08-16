package emulator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"cloud.google.com/go/iam/apiv1/iampb"

	"github.com/go-chi/chi"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"github.com/rs/zerolog"
)

type Emulator struct {
	endoint    string
	testServer *server.TestServer
	emulator   *server.Server
	log        zerolog.Logger
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

func (e *Emulator) EnableMock(debugRequest bool, log zerolog.Logger, mocks ...*EndpointMock) {
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

				log.Info().Msgf("request: \n%s", string(request))
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

			log.Info().Msgf("request: \n%s", string(request))
		}

		handler.ServeHTTP(w, r)
	})

	e.emulator.Handler = router
}

func (e *Emulator) Cleanup() {
	e.testServer.Close()
}

func (e *Emulator) Endpoint() string {
	if len(e.endoint) > 0 {
		return e.endoint
	}

	return e.testServer.URL
}

func (e *Emulator) WithProject(projectID string, datasets ...*Dataset) {
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

			t.Columns = append(t.Columns, ds.Columns...)
			d.Tables = append(d.Tables, t)
		}

		p.Datasets = append(p.Datasets, d)
	}

	e.WithSource(p.ID, server.StructSource(p))
}

func (e *Emulator) WithSource(projectID string, source server.Source) {
	err := e.emulator.Load(source)
	if err != nil {
		e.log.Fatal().Err(err).Msg("initializing bigquery emulator")
	}

	if err := e.emulator.SetProject(projectID); err != nil {
		e.log.Fatal().Err(err).Msg("setting project")
	}
}

func (e *Emulator) TestServer() {
	e.testServer = e.emulator.TestServer()
}

func (e *Emulator) Serve(ctx context.Context, httpAddr, grpcAddr string) error {
	e.endoint = httpAddr

	err := e.emulator.Serve(ctx, httpAddr, grpcAddr)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	return nil
}

func New(log zerolog.Logger) *Emulator {
	s, err := server.New(server.TempStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("creating bigquery emulator")
	}

	return &Emulator{
		emulator: s,
		log:      log,
	}
}

type PolicyMock struct {
	log      zerolog.Logger
	policies map[string]*iampb.Policy
}

func NewPolicyMock(log zerolog.Logger) *PolicyMock {
	return &PolicyMock{
		log:      log,
		policies: make(map[string]*iampb.Policy),
	}
}

func (p *PolicyMock) Mocks() []*EndpointMock {
	return []*EndpointMock{
		{
			Method:  http.MethodPost,
			Path:    "/projects/{project}/datasets/{dataset}/tables/{table}:getIamPolicy",
			Handler: p.GetPolicy(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/projects/{project}/datasets/{dataset}/tables/{table}:setIamPolicy",
			Handler: p.SetPolicy(),
		},
	}
}

func (p *PolicyMock) GetPolicy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := chi.URLParam(r, "project")
		dataset := chi.URLParam(r, "dataset")
		table := chi.URLParam(r, "table")

		key := fmt.Sprintf("%s/%s/%s", project, dataset, table)

		policy := &iampb.Policy{}
		_, hasKey := p.policies[key]
		if hasKey {
			policy = p.policies[key]
		}

		err := json.NewEncoder(w).Encode(policy)
		if err != nil {
			p.log.Error().Err(err).Msg("Failed to encode policy")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (p *PolicyMock) SetPolicy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := chi.URLParam(r, "project")
		dataset := chi.URLParam(r, "dataset")
		table := chi.URLParam(r, "table")

		policyRequest := &iampb.SetIamPolicyRequest{}

		err := json.NewDecoder(r.Body).Decode(policyRequest)
		if err != nil {
			p.log.Error().Err(err).Msg("Failed to decode policy")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		key := fmt.Sprintf("%s/%s/%s", project, dataset, table)
		p.policies[key] = policyRequest.Policy

		// If the header is StatusNoContent, the body is ignored by the google client
		// otherwise it will try to parse the body
		w.WriteHeader(http.StatusNoContent)
	}
}

func DatasetTableIAMPolicyGetMock(project, dataset, table string, log zerolog.Logger, policy *iampb.Policy) *EndpointMock {
	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(policy)
		if err != nil {
			log.Error().Err(err).Msg("Failed to encode policy")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	return &EndpointMock{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/projects/%s/datasets/%s/tables/%s:getIamPolicy", project, dataset, table),
		Handler: handlerFn,
	}
}

func DatasetTableIAMPolicySetMock(project, dataset, table string, log zerolog.Logger, into *iampb.SetIamPolicyRequest) *EndpointMock {
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
		Path:    fmt.Sprintf("/projects/%s/datasets/%s/tables/%s:setIamPolicy", project, dataset, table),
		Handler: handlerFn,
	}
}
