package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/iam/v1"

	"github.com/go-chi/chi"
	"github.com/google/go-cmp/cmp"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type metabaseSetupBody struct {
	Token string        `json:"token"`
	User  metabaseUser  `json:"user"`
	Prefs metabasePrefs `json:"prefs"`
}

type metabaseUser struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type metabasePrefs struct {
	AllowTracking bool   `json:"allow_tracking"`
	SiteName      string `json:"site_name"`
}

type CleanupFn func()

type containers struct {
	t         *testing.T
	log       zerolog.Logger
	pool      *dockertest.Pool
	resources []*dockertest.Resource
}

// Cleanup may be deferred in a test function to ensure that all resources are purged.
func (c *containers) Cleanup() {
	for _, r := range c.resources {
		if err := c.pool.Purge(r); err != nil {
			c.log.Warn().Err(err).Msg("purging resources")
		}
	}
}

type PostgresConfig struct {
	User     string
	Password string
	Database string

	// HostPort is populated after the container is started.
	HostPort string
}

func (c *PostgresConfig) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", c.User, c.Password, c.HostPort, c.Database)
}

func NewPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		User:     "nada-backend",
		Password: "supersecret",
		Database: "nada",
	}
}

func (c *containers) RunPostgres(cfg *PostgresConfig) *PostgresConfig {
	var db *sql.DB

	resource, err := c.pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
			fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
			fmt.Sprintf("POSTGRES_DB=%s", cfg.Database),
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		c.t.Fatalf("starting postgres container: %s", err)
	}

	cfg.HostPort = resource.GetHostPort("5432/tcp")
	c.log.Info().Msgf("Postgres is configured with url: %s", cfg.ConnectionURL())

	c.pool.MaxWait = 120 * time.Second
	c.resources = append(c.resources, resource)

	if err = c.pool.Retry(func() error {
		db, err = sql.Open("postgres", cfg.ConnectionURL())
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		c.t.Fatalf("could not connect to postgres: %s", err)
	}

	return cfg
}

type MetabaseConfig struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	SiteName  string

	// PremiumEmbeddingToken is populated from the environment.
	PremiumEmbeddingToken string

	// HostPort is populated after the container is started.
	HostPort string
}

func (m *MetabaseConfig) SessionPropertiesURL() string {
	return fmt.Sprintf("http://%s/api/session/properties", m.HostPort)
}

func (m *MetabaseConfig) SetupURL() string {
	return fmt.Sprintf("http://%s/api/setup", m.HostPort)
}

func (m *MetabaseConfig) ConnectionURL() string {
	return fmt.Sprintf("http://%s", m.HostPort)
}

func (m *MetabaseConfig) SetupBody(token string) *metabaseSetupBody {
	return &metabaseSetupBody{
		Token: token,
		User: metabaseUser{
			Email:     m.Email,
			Password:  m.Password,
			FirstName: m.FirstName,
			LastName:  m.LastName,
		},
		Prefs: metabasePrefs{
			AllowTracking: false,
			SiteName:      m.SiteName,
		},
	}
}

func NewMetabaseConfig() *MetabaseConfig {
	return &MetabaseConfig{
		FirstName:             "Nada",
		LastName:              "Backend",
		Email:                 "nada@nav.no",
		Password:              "superdupersecret1",
		SiteName:              "Nada Backend",
		PremiumEmbeddingToken: os.Getenv("MB_PREMIUM_EMBEDDING_TOKEN"),
	}
}

func (c *containers) RunMetabase(cfg *MetabaseConfig) *MetabaseConfig {
	metabaseVersion, err := os.ReadFile("../../.metabase_version")
	if err != nil {
		c.t.Fatalf("loading metabase version: %s", err)
	}

	c.log.Info().Msgf("Metabase version: %s", metabaseVersion)

	resource, err := c.pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "europe-north1-docker.pkg.dev/nada-prod-6977/nada-north/metabase-patched",
		Tag:        strings.TrimSpace(string(metabaseVersion)),
		Env: []string{
			"MB_DB_TYPE=h2",
			"MB_ENABLE_PASSWORD_LOGIN=true",
			fmt.Sprintf("MB_PREMIUM_EMBEDDING_TOKEN=%s", cfg.PremiumEmbeddingToken),
		},
		Platform: "linux/amd64",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		c.t.Fatalf("starting metabase container: %s", err)
	}

	cfg.HostPort = resource.GetHostPort("3000/tcp")
	c.log.Info().Msgf("Metabase is configured with url: %s", cfg.ConnectionURL())

	c.pool.MaxWait = 2 * time.Minute
	c.resources = append(c.resources, resource)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Exponential backoff-retry to connect to Metabase instance
	if err := c.pool.Retry(func() error {
		resp, err := client.Get(cfg.SessionPropertiesURL())
		if err != nil {
			c.log.Warn().Err(err).Msg("could not get session properties")
			return err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != 200 {
			return fmt.Errorf("server not ready")
		}

		return nil
	}); err != nil {
		c.t.Fatalf("could not connect to metabase: %s", err)
	}

	resp, err := client.Get(cfg.SessionPropertiesURL())
	if err != nil {
		c.t.Fatalf("could not get session properties: %s", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	token := struct {
		SetupToken string `json:"setup-token"`
	}{}
	Unmarshal(c.t, resp.Body, &token)

	resp, err = client.Post(cfg.SetupURL(), "application/json", bytes.NewReader(Marshal(c.t, cfg.SetupBody(token.SetupToken))))
	if err != nil {
		c.t.Fatalf("could not setup metabase: %s", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		c.t.Fatalf("could not setup metabase: %s", resp.Status)
	}

	return cfg
}

func NewContainers(t *testing.T, log zerolog.Logger) *containers {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("connecting to Docker: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		t.Fatalf("pinging Docker: %s", err)
	}

	return &containers{
		t:         t,
		log:       log,
		pool:      pool,
		resources: nil,
	}
}

func Marshal(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling: %s", err)
	}

	return b
}

func Unmarshal(t *testing.T, r io.Reader, v interface{}) {
	t.Helper()

	d, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading: %s", err)
	}

	err = json.Unmarshal(d, v)
	if err != nil {
		t.Fatalf("unmarshaling: %s", err)
	}
}

type TestRunner interface {
	Post(input any, path string, params ...string) TestRunnerStatus
	Get(path string, params ...string) TestRunnerStatus
	Put(input any, path string, params ...string) TestRunnerStatus
	Delete(path string, params ...string) TestRunnerStatus
	Headers(headers map[string]string) TestRunner
	Send(r *http.Request) TestRunnerStatus
}

type TestRunnerStatus interface {
	Debug(out io.Writer) TestRunnerStatus
	HasStatusCode(code int) TestRunnerEnder
}

type TestRunnerEnder interface {
	Body() string
	Value(into any)
	Expect(expect, into any, opts ...cmp.Option)
}

type testRunner struct {
	t *testing.T
	s *httptest.Server

	headers  map[string]string
	response *http.Response
}

func (r *testRunner) HasStatusCode(code int) TestRunnerEnder {
	r.t.Helper()

	if r.response.StatusCode != code {
		r.t.Errorf("expected status code %d, got %d", code, r.response.StatusCode)
	}

	return r
}

func (r *testRunner) Debug(out io.Writer) TestRunnerStatus {
	r.t.Helper()

	data, err := httputil.DumpRequest(r.response.Request, true)
	if err != nil {
		r.t.Fatalf("dumping request: %s", err)
	}

	_, err = io.Copy(out, bytes.NewReader(data))
	if err != nil {
		r.t.Fatalf("writing request: %s", err)
	}

	data, err = httputil.DumpResponse(r.response, true)
	if err != nil {
		r.t.Fatalf("dumping response: %s", err)
	}

	_, err = io.Copy(out, bytes.NewReader(data))
	if err != nil {
		r.t.Fatalf("writing response: %s", err)
	}

	return r
}

func (r *testRunner) Expect(expect, into any, opts ...cmp.Option) {
	r.t.Helper()

	Unmarshal(r.t, r.response.Body, into)
	diff := cmp.Diff(expect, into, opts...)
	if diff != "" {
		r.t.Errorf("unexpected response: %s", diff)
	}
}

func (r *testRunner) Value(into any) {
	r.t.Helper()

	Unmarshal(r.t, r.response.Body, into)
}

func (r *testRunner) Body() string {
	r.t.Helper()

	data, err := io.ReadAll(r.response.Body)
	if err != nil {
		r.t.Fatalf("reading body: %s", err)
	}

	return string(data)
}

func (r *testRunner) parseQueryParams(params ...string) string {
	r.t.Helper()

	if len(params) == 0 {
		return ""
	}

	if len(params)%2 != 0 {
		r.t.Fatalf("invalid number of query parameters")
	}

	var p []string
	for i := 0; i < len(params); i += 2 {
		p = append(p, fmt.Sprintf("%s=%s", params[i], params[i+1]))
	}

	return "?" + strings.Join(p, "&")
}

func (r *testRunner) buildURL(path string, params ...string) string {
	return fmt.Sprintf("%s%s%s", r.s.URL, path, r.parseQueryParams(params...))
}

func (r *testRunner) Send(req *http.Request) TestRunnerStatus {
	r.t.Helper()

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		r.t.Fatalf("sending request: %s", err)
	}

	r.response = response

	return r
}

func (r *testRunner) Headers(headers map[string]string) TestRunner {
	r.t.Helper()

	r.headers = headers

	return r
}

func (r *testRunner) Get(path string, params ...string) TestRunnerStatus {
	r.t.Helper()

	url := r.buildURL(path, params...)
	r.response = SendRequest(r.t, http.MethodGet, url, nil, r.headers)

	return r
}

func (r *testRunner) Put(input any, path string, params ...string) TestRunnerStatus {
	r.t.Helper()

	url := r.buildURL(path, params...)
	r.response = SendRequest(r.t, http.MethodPut, url, bytes.NewReader(Marshal(r.t, input)), r.headers)

	return r
}

func (r *testRunner) Delete(path string, params ...string) TestRunnerStatus {
	r.t.Helper()

	url := r.buildURL(path, params...)
	r.response = SendRequest(r.t, http.MethodDelete, url, nil, r.headers)

	return r
}

func (r *testRunner) Post(input any, path string, params ...string) TestRunnerStatus {
	r.t.Helper()

	url := r.buildURL(path, params...)
	r.response = SendRequest(r.t, http.MethodPost, url, bytes.NewReader(Marshal(r.t, input)), r.headers)

	return r
}

func NewTester(t *testing.T, s *httptest.Server) *testRunner {
	return &testRunner{
		t:       t,
		s:       s,
		headers: map[string]string{},
	}
}

func SendRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("creating request: %s", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %s", err)
	}

	return resp
}

func strToStrPtr(s string) *string {
	return &s
}

func injectUser(user *service.User) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r.WithContext(auth.SetUser(r.Context(), user)))
		})
	}
}

func TestRouter(log zerolog.Logger) chi.Router {
	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Error().Str("method", r.Method).Str("path", r.URL.Path).Msg("not found")
		w.WriteHeader(http.StatusNotFound)
	})

	return r
}

func CreateMultipartFormRequest(t *testing.T, method, path string, files map[string]string, objects map[string]string, headers map[string]string) *http.Request {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	for path, data := range files {
		part, err := writer.CreateFormFile(path, filepath.Base(path))
		assert.NoError(t, err)
		_, err = part.Write([]byte(data))
		assert.NoError(t, err)
	}

	for name, data := range objects {
		part, err := writer.CreateFormField(name)
		assert.NoError(t, err)
		_, err = part.Write([]byte(data))
		assert.NoError(t, err)
	}

	writer.Close()

	req, err := http.NewRequest(method, path, &b)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}

func GetFreePort(t *testing.T) int {
	t.Helper()

	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("get free port, resolving: %s", err)
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		t.Fatalf("get free port, listening: %s", err)
	}

	defer func(l *net.TCPListener) {
		_ = l.Close()
	}(l)

	return l.Addr().(*net.TCPAddr).Port
}

func ContainsCollectionWithName(collections []*service.MetabaseCollection, expectedName string) bool {
	for _, collection := range collections {
		if collection.Name == expectedName {
			return true
		}
	}

	return false
}

func ContainsPermissionGroupWithNamePrefix(permissionGroups []service.MetabasePermissionGroup, prefix string) bool {
	for _, permissionGroup := range permissionGroups {
		if strings.HasPrefix(permissionGroup.Name, prefix) {
			return true
		}
	}

	return false
}

func ContainsServiceAccount(serviceAccounts map[string]*iam.ServiceAccount, prefix, postfix string) bool {
	for _, sa := range serviceAccounts {
		if strings.HasPrefix(sa.Email, prefix) && strings.HasSuffix(sa.Email, postfix) {
			return true
		}
	}

	return false
}
