package main

type Config struct {
	BindAddress                     string
	DBConnectionDSN                 string
	LogLevel                        string
	OAuth2                          OAuth2Config
	TeamProjectsOutputURL           string
	TeamsToken                      string
	Hostname                        string
	CookieSecret                    string
	MockAuth                        bool
	SkipMetadataSync                bool
	ServiceAccountFile              string
	GoogleAdminImpersonationSubject string
	TeamkatalogenURL                string
	MetabaseServiceAccountFile      string
	MetabaseUsername                string
	MetabasePassword                string
	MetabaseAPI                     string
	SlackUrl                        string
	SlackToken                      string
	PollyURL                        string
	DBMaxIdleConn                   int
	DBMaxOpenConn                   int
	QuartoStorageBucketName         string
	ConsoleAPIKey                   string
	ConsoleURL                      string
	AmplitudeAPIKey                 string
}

type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

func DefaultConfig() Config {
	return Config{
		BindAddress:           ":8080",
		LogLevel:              "info",
		TeamkatalogenURL:      "https://teamkatalog-api.intern.nav.no",
		PollyURL:              "https://polly.prod-fss-pub.nais.io/process",
		TeamProjectsOutputURL: "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/prod-output.json",
		ConsoleURL:            "https://teams.nav.cloud.nais.io",
	}
}
