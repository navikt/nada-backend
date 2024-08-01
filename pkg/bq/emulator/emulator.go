package emulator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"cloud.google.com/go/iam/apiv1/iampb"

	"github.com/go-chi/chi"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
)

type Emulator struct {
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

func (e *Emulator) Serve(ctx context.Context, httpPort, grpcPort string) error {
	err := e.emulator.Serve(
		ctx,
		fmt.Sprintf("0.0.0.0:%s", httpPort),
		fmt.Sprintf("0.0.0.0:%s", grpcPort),
	)
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

func PolicyMocksFromDataYAML(path string, log zerolog.Logger) ([]*EndpointMock, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v struct {
		Projects []*types.Project `yaml:"projects" validate:"required"`
	}

	dec := yaml.NewDecoder(
		bytes.NewBuffer(content),
		yaml.Strict(),
	)

	if err := dec.Decode(&v); err != nil {
		return nil, errors.New(yaml.FormatError(err, false, true))
	}

	var mocks []*EndpointMock
	for _, project := range v.Projects {
		for _, dataset := range project.Datasets {
			for _, table := range dataset.Tables {
				mocks = append(mocks, DatasetTableIAMPolicyGetMock(project.ID, dataset.ID, table.ID, log, &iampb.Policy{}))
				mocks = append(mocks, DatasetTableIAMPolicySetMock(project.ID, dataset.ID, table.ID, log, &iampb.SetIamPolicyRequest{}))
			}
		}
	}

	return mocks, nil
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
