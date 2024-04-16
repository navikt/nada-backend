package config

var Conf = DefaultConfig()

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
	StoryBucketName                 string
	ConsoleAPIKey                   string
	ConsoleURL                      string
	AmplitudeAPIKey                 string
	CentralDataProject              string
	PseudoDataset                   string
	NadaTokenCreds                  string
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
		TeamkatalogenURL:      "http://team-catalog-backend.org.svc.cluster.local",
		PollyURL:              "http://behandlingskatalog-backend.teamdatajegerne.svc.cluster.local/process",
		TeamProjectsOutputURL: "https://raw.githubusercontent.com/nais/teams/master/gcp-projects/prod-output.json",
		ConsoleURL:            "https://console.nav.cloud.nais.io",
	}
}
