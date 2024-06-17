package core

import (
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/api"
	"github.com/navikt/nada-backend/pkg/service/core/storage"
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
}

func NewServices(
	cfg config.Config,
	stores *storage.Stores,
	clients *api.Clients,
) (*Services, error) {
	// FIXME: not sure about this..
	mbSa, mbSaEmail, err := cfg.Metabase.LoadFromCredentialsPath()
	if err != nil {
		return nil, err
	}

	return &Services{
		AccessService: NewAccessService(
			clients.SlackAPI,
			stores.PollyStorage,
			stores.AccessStorage,
		),
		BigQueryService: NewBigQueryService(
			stores.BigQueryStorage,
			clients.BigQueryAPI,
		),
		DataProductService: NewDataProductsService(
			stores.DataProductsStorage,
		),
		InsightProductService: NewInsightProductService(
			stores.InsightProductStorage,
		),
		JoinableViewService: NewJoinableViewsService(
			stores.JoinableViewsStorage,
		),
		KeyWordService: NewKeywordsService(
			stores.KeyWordStorage,
		),
		MetaBaseService: NewMetabaseService(
			cfg.GCP.Project,
			mbSa,
			mbSaEmail,
			clients.MetaBaseAPI,
			clients.BigQueryAPI,
			clients.ServiceAccountAPI,
			stores.ThirdPartyMappingStorage,
			stores.MetaBaseStorage,
			stores.BigQueryStorage,
			stores.DataProductsStorage,
			stores.AccessStorage,
		),
		PollyService: NewPollyService(
			stores.PollyStorage,
			clients.PollyAPI,
		),
		ProductAreaService: NewProductAreaService(
			stores.ProductAreaStorage,
		),
		SearchService: NewSearchService(
			stores.SearchStorage,
		),
		SlackService: NewSlackService(
			clients.SlackAPI,
		),
		StoryService: NewStoryService(
			stores.StoryStorage,
		),
		TeamKatalogenService: NewTeamKatalogenService(
			clients.TeamKatalogenAPI,
		),
		TokenService: NewTokenService(
			stores.TokenStorage,
		),
		UserService: NewUserService(),
	}, nil
}
