package config_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/navikt/nada-backend/pkg/config/v2"

	"github.com/google/go-cmp/cmp"

	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "update golden files")

func newFakeConfig() config.Config {
	return config.Config{
		Oauth: config.Oauth{
			ClientID:     "fake_client_id",
			ClientSecret: "fake_client_secret",
			TenantID:     "fake_tenant_id",
			RedirectURL:  "http://localhost:8080/auth/callback",
		},
		Metabase: config.Metabase{
			Username:         "fake_username",
			Password:         "fake_password",
			APIURL:           "http://localhost:3000/api",
			GCPProject:       "some-gcp-project",
			CredentialsPath:  "/some/path",
			DatabasesBaseURL: "http://localhost:3000",
			BigQueryDatabase: config.MetabaseBigQueryDatabase{
				APIEndpointOverride: "http://localhost:3000",
				DisableAuth:         true,
			},
			MappingDeadlineSec:  600,
			MappingFrequencySec: 600,
		},
		CrossTeamPseudonymization: config.CrossTeamPseudonymization{
			GCPProjectID: "some-project",
			GCPRegion:    "eu-north1",
		},
		GCS: config.GCS{
			Endpoint:          "http://localhost:9090",
			StoryBucketName:   "some-bucket",
			CentralGCPProject: "central-project",
		},
		BigQuery: config.BigQuery{
			Endpoint:                          "http://localhost:7070",
			TeamProjectPseudoViewsDatasetName: "some-dataset",
			GCPRegion:                         "eu-north1",
			CentralGCPProject:                 "central-project",
		},
		Slack: config.Slack{
			Token:      "fake_token",
			WebhookURL: "http://localhost:8080/webhook",
		},
		Server: config.Server{
			Hostname: "localhost",
			Address:  "127.0.0.1",
			Port:     "8080",
		},
		Postgres: config.Postgres{
			UserName:     "fake_username",
			Password:     "fake_password",
			Host:         "http://localhost",
			Port:         "5432",
			DatabaseName: "something",
			SSLMode:      "disable",
			Configuration: config.PostgresConfiguration{
				MaxIdleConnections: 10,
				MaxOpenConnections: 5,
			},
		},
		TeamsCatalogue: config.TeamsCatalogue{
			APIURL: "http://localhost:8080/api",
		},
		TreatmentCatalogue: config.TreatmentCatalogue{
			APIURL:     "http://localhost:8080/api",
			PurposeURL: "http://localhost:8080/api/purpose",
		},
		GoogleGroups: config.GoogleGroups{
			ImpersonationSubject: "something@example.com",
			CredentialsFile:      "/some/secret/path",
		},
		Cookies: config.Cookies{
			Redirect: config.CookieSettings{
				Name:     "redirect",
				MaxAge:   3600,
				Path:     "some/path",
				Domain:   "localhost",
				SameSite: "Lax",
				Secure:   false,
				HttpOnly: true,
			},
			OauthState: config.CookieSettings{
				Name:     "auth",
				MaxAge:   3600,
				Path:     "some/path",
				Domain:   "localhost",
				SameSite: "Lax",
				Secure:   false,
				HttpOnly: true,
			},
			Session: config.CookieSettings{
				Name:     "session",
				MaxAge:   3600,
				Path:     "some/path",
				Domain:   "localhost",
				SameSite: "Lax",
				Secure:   false,
				HttpOnly: true,
			},
		},
		NaisConsole: config.NaisConsole{
			APIKey: "fake_api_key",
			APIURL: "http://localhost:8080/api",
		},
		API: config.API{
			AuthToken: "fake_token",
		},
		ServiceAccount: config.ServiceAccount{
			EndpointOverride: "http://localhost:8086",
			DisableAuth:      true,
		},
		EmailSuffix:                    "@nav.no",
		NaisClusterName:                "dev-gcp",
		KeywordsAdminGroup:             "nada@nav.no",
		AllUsersGroup:                  "group:all-users@nav.no",
		LoginPage:                      "http://localhost:8080/",
		AmplitudeAPIKey:                "fake_key",
		LogLevel:                       "info",
		CacheDurationSeconds:           60,
		TeamProjectsUpdateDelaySeconds: 120,
		StoryCreateIgnoreMissingTeam:   false,
		Debug:                          false,
	}
}

func updateGoldenFiles(t *testing.T, filePath string, cfg config.Config) []byte {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Errorf("marshal config: %v", err)
	}

	err = os.WriteFile(filePath, data, 0o600)
	if err != nil {
		t.Errorf("write golden file: %v", err)
	}

	return data
}

func TestValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		config    config.Config
		expectErr bool
	}{
		{
			name:      "Valid config",
			config:    newFakeConfig(),
			expectErr: false,
		},
		{
			name: "Invalid config",
			config: func() config.Config {
				cfg := newFakeConfig()
				cfg.Oauth.ClientID = ""

				return cfg
			}(),
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if err != nil && !tc.expectErr {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil && tc.expectErr {
				t.Errorf("expected error, got none")
			}
		})
	}
}

func TestLoad(t *testing.T) {
	if *update {
		t.Log("Updating golden files")
		updateGoldenFiles(t, "testdata/config.yaml", newFakeConfig())
		t.Log("Done updating golden files")

		return
	}

	testCases := []struct {
		name      string
		config    string
		path      string
		envPrefix string
		loader    config.Loader
		binder    config.Binder
		envs      map[string]string
		expect    config.Config
		expectErr bool
	}{
		{
			name:      "Standard config",
			config:    "config",
			path:      "testdata",
			loader:    config.NewFileSystemLoader(),
			expect:    newFakeConfig(),
			expectErr: false,
		},
		{
			name:   "Standard config with env overrides",
			config: "config",
			path:   "testdata",
			loader: config.NewFileSystemLoader(),
			expect: func() config.Config {
				cfg := newFakeConfig()
				cfg.Server.Address = "example.com"

				return cfg
			}(),
			envs: map[string]string{
				"SERVER_ADDRESS": "example.com",
			},
		},
		{
			name:      "Standard config with env prefix overrides",
			config:    "config",
			path:      "testdata",
			envPrefix: "nada",
			loader:    config.NewFileSystemLoader(),
			expect: func() config.Config {
				cfg := newFakeConfig()
				cfg.Server.Address = "example.com"

				return cfg
			}(),
			envs: map[string]string{
				"NADA_SERVER_ADDRESS": "example.com",
			},
		},
		{
			name:      "Standard config with env overrides and binder",
			config:    "config",
			path:      "testdata",
			envPrefix: "nada",
			loader:    config.NewFileSystemLoader(),
			binder: config.NewEnvBinder(map[string]string{
				"SOME_RANDOM_SERVER_ADDRESS": "server.address",
			}),
			expect: func() config.Config {
				cfg := newFakeConfig()
				cfg.Server.Address = "this.example.com"

				return cfg
			}(),
			envs: map[string]string{
				"SOME_RANDOM_SERVER_ADDRESS": "this.example.com",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			cfg, err := tc.loader.Load(tc.config, tc.path, tc.envPrefix, tc.binder)
			if err != nil && !tc.expectErr {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil && tc.expectErr {
				t.Errorf("expected error, got none")
			}

			if !tc.expectErr {
				if diff := cmp.Diff(tc.expect, cfg); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func getWorkingDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("get working dir: %v", err)
	}

	return wd
}

func TestProcessConfigPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		path      string
		expect    config.FileParts
		expectErr bool
	}{
		{
			name: "Valid config path",
			path: "testdata/config.yaml",
			expect: config.FileParts{
				FileName: "config",
				Path:     filepath.Join(getWorkingDir(t), "testdata"),
			},
		},
		{
			name:      "Invalid extension",
			path:      "testdata/config.json",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.ProcessConfigPath(tc.path)
			if err != nil && !tc.expectErr {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil && tc.expectErr {
				t.Errorf("expected error, got none")
			}

			if !tc.expectErr {
				if diff := cmp.Diff(tc.expect, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
