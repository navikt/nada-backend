package main

type Config struct {
	BindAddress                     string
	DBConnectionDSN                 string
	LogLevel                        string
	OAuth2                          OAuth2Config
	DevTeamProjectsOutputURL        string
	ProdTeamProjectsOutputURL       string
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
	PollyURL                        string
}

type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

func DefaultConfig() Config {
	return Config{
		BindAddress:               ":8080",
		LogLevel:                  "info",
		DevTeamProjectsOutputURL:  "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/dev-output.json",
		ProdTeamProjectsOutputURL: "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/prod-output.json",
		TeamkatalogenURL:          "https://teamkatalog-api.prod-fss-pub.nais.io",
		PollyURL:                  "https://polly.prod-fss-pub.nais.io/process",
	}
}
