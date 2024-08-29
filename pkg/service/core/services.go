package core

import (
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
	"github.com/rs/zerolog"
)

type Services struct {
	AccessService         service.AccessService
	BigQueryService       service.BigQueryService
	DataProductService    service.DataProductsService
	InsightProductService service.InsightProductService
	JoinableViewService   service.JoinableViewsService
	KeyWordService        service.KeywordsService
	MetaBaseService       service.MetabaseService
	PollyService          service.PollyService
	ProductAreaService    service.ProductAreaService
	SearchService         service.SearchService
	SlackService          service.SlackService
	StoryService          service.StoryService
	TeamKatalogenService  service.TeamKatalogenService
	TokenService          service.TokenService
	UserService           service.UserService
	NaisConsoleService    service.NaisConsoleService
}

func NewServices(
	cfg config.Config,
	stores *storage.Stores,
	clients *api.Clients,
	log zerolog.Logger,
) (*Services, error) {
	// FIXME: not sure about this..
	mbSa, mbSaEmail, err := cfg.Metabase.LoadFromCredentialsPath()
	if err != nil {
		return nil, err
	}

	return &Services{
		AccessService: NewAccessService(
			cfg.Server.Hostname,
			clients.SlackAPI,
			stores.PollyStorage,
			stores.AccessStorage,
			stores.DataProductsStorage,
			stores.BigQueryStorage,
			stores.JoinableViewsStorage,
			clients.BigQueryAPI,
		),
		BigQueryService: NewBigQueryService(
			stores.BigQueryStorage,
			clients.BigQueryAPI,
			stores.DataProductsStorage,
		),
		DataProductService: NewDataProductsService(
			stores.DataProductsStorage,
			stores.BigQueryStorage,
			clients.BigQueryAPI,
			stores.NaisConsoleStorage,
			cfg.AllUsersGroup,
		),
		InsightProductService: NewInsightProductService(
			stores.InsightProductStorage,
		),
		JoinableViewService: NewJoinableViewsService(
			stores.JoinableViewsStorage,
			stores.AccessStorage,
			stores.DataProductsStorage,
			clients.BigQueryAPI,
			stores.BigQueryStorage,
		),
		KeyWordService: NewKeywordsService(
			stores.KeyWordStorage,
			cfg.KeywordsAdminGroup,
		),
		MetaBaseService: NewMetabaseService(
			cfg.Metabase.GCPProject,
			mbSa,
			mbSaEmail,
			cfg.AllUsersGroup,
			clients.MetaBaseAPI,
			clients.BigQueryAPI,
			clients.ServiceAccountAPI,
			stores.ThirdPartyMappingStorage,
			stores.MetaBaseStorage,
			stores.BigQueryStorage,
			stores.DataProductsStorage,
			stores.AccessStorage,
			log.With().Str("service", "metabase").Logger(),
		),
		PollyService: NewPollyService(
			stores.PollyStorage,
			clients.PollyAPI,
		),
		ProductAreaService: NewProductAreaService(
			stores.ProductAreaStorage,
			stores.DataProductsStorage,
			stores.InsightProductStorage,
			stores.StoryStorage,
		),
		SearchService: NewSearchService(
			stores.SearchStorage,
			stores.StoryStorage,
			stores.DataProductsStorage,
		),
		SlackService: NewSlackService(
			clients.SlackAPI,
		),
		StoryService: NewStoryService(
			stores.StoryStorage,
			clients.TeamKatalogenAPI,
			clients.StoryAPI,
			cfg.StoryCreateIgnoreMissingTeam,
		),
		TeamKatalogenService: NewTeamKatalogenService(
			clients.TeamKatalogenAPI,
		),
		TokenService: NewTokenService(
			stores.TokenStorage,
		),
		UserService: NewUserService(
			stores.AccessStorage,
			stores.TokenStorage,
			stores.StoryStorage,
			stores.DataProductsStorage,
			stores.InsightProductStorage,
			stores.NaisConsoleStorage,
			log,
		),
		NaisConsoleService: NewNaisConsoleService(
			stores.NaisConsoleStorage,
			clients.NaisConsoleAPI,
		),
	}, nil
}
