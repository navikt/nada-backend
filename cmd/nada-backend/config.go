package main

type Config struct {
	BindAddress               string
	DBConnectionDSN           string
	LogLevel                  string
	OAuth2                    OAuth2Config
	TeamsURL                  string
	DevTeamProjectsOutputURL  string
	ProdTeamProjectsOutputURL string
	TeamsToken                string
	Hostname                  string
	CookieSecret              string
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
		TeamsURL:                  "https://raw.githubusercontent.com/navikt/teams/main/teams.json",
		DevTeamProjectsOutputURL:  "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/dev-output.json",
		ProdTeamProjectsOutputURL: "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/prod-output.json",
	}
}
