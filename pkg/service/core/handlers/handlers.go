package handlers

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"net/http"
)

type Endpoints struct {
	GetGCSObject                       http.HandlerFunc
	CreateStoryHTTP                    http.HandlerFunc
	UpdateStoryHTTP                    http.HandlerFunc
	AppendStoryHTTP                    http.HandlerFunc
	GetAllTeamTokens                   http.HandlerFunc
	GetDataProduct                     http.HandlerFunc
	CreateDataProduct                  http.HandlerFunc
	DeleteDataProduct                  http.HandlerFunc
	UpdateDataProduct                  http.HandlerFunc
	GetDatasetsMinimal                 http.HandlerFunc
	GetDataset                         http.HandlerFunc
	MapDataset                         http.HandlerFunc
	CreateDataset                      http.HandlerFunc
	UpdateDataset                      http.HandlerFunc
	DeleteDataset                      http.HandlerFunc
	GetAccessiblePseudoDatasetsForUser http.HandlerFunc
	GetAccessRequests                  http.HandlerFunc
	ProcessAccessRequest               http.HandlerFunc
	CreateAccessRequest                http.HandlerFunc
	DeleteAccessRequest                http.HandlerFunc
	UpdateAccessRequest                http.HandlerFunc
	GetProductAreas                    http.HandlerFunc
	GetProductAreaWithAssets           http.HandlerFunc
	GetBigQueryColumns                 http.HandlerFunc
	GetBigQueryTables                  http.HandlerFunc
	GetBigQueryDatasets                http.HandlerFunc
	SyncBigQueryTables                 http.HandlerFunc
	Search                             http.HandlerFunc
	RotateNadaToken                    http.HandlerFunc
	GetUserData                        http.HandlerFunc
	IsValidSlackChannel                http.HandlerFunc
	GetStoryMetadata                   http.HandlerFunc
	CreateStory                        http.HandlerFunc
	UpdateStory                        http.HandlerFunc
	DeleteStory                        http.HandlerFunc
	GrantAccessToDataset               http.HandlerFunc
	RevokeAccessToDataset              http.HandlerFunc
	CreateJoinableViews                http.HandlerFunc
	GetJoinableViewsForUser            http.HandlerFunc
	GetJoinableView                    http.HandlerFunc
	GetInsightProduct                  http.HandlerFunc
	CreateInsightProduct               http.HandlerFunc
	UpdateInsightProduct               http.HandlerFunc
	DeleteInsightProduct               http.HandlerFunc
	SearchTeamKatalogen                http.HandlerFunc
	SearchPolly                        http.HandlerFunc
	GetKeywordsListSortedByPopularity  http.HandlerFunc
	UpdateKeywords                     http.HandlerFunc
}

func NewEndpoints(h *Handlers, log zerolog.Logger) *Endpoints {
	return &Endpoints{
		// Story endpoints
		GetGCSObject:     h.StoryHandler.GetGCSObject,
		CreateStoryHTTP:  h.StoryHandler.CreateStoryHTTP,
		UpdateStoryHTTP:  h.StoryHandler.UpdateStoryHTTP,
		AppendStoryHTTP:  h.StoryHandler.AppendStoryHTTP,
		GetStoryMetadata: transport.For(h.StoryHandler.GetStoryMetadata).Build(log),
		CreateStory:      transport.For(h.StoryHandler.CreateStory).Build(log),
		UpdateStory:      transport.For(h.StoryHandler.UpdateStory).RequestFromJSON().Build(log),
		DeleteStory:      transport.For(h.StoryHandler.DeleteStory).Build(log),

		// Token endpoints
		GetAllTeamTokens: h.TokenHandler.GetAllTeamTokens,
		RotateNadaToken:  transport.For(h.TokenHandler.RotateNadaToken).Build(log),

		// Data product endpoints
		GetDataProduct:     transport.For(h.DataProductsHandler.GetDataProduct).Build(log),
		CreateDataProduct:  transport.For(h.DataProductsHandler.CreateDataProduct).RequestFromJSON().Build(log),
		DeleteDataProduct:  transport.For(h.DataProductsHandler.DeleteDataProduct).Build(log),
		UpdateDataProduct:  transport.For(h.DataProductsHandler.UpdateDataProduct).RequestFromJSON().Build(log),
		GetDatasetsMinimal: transport.For(h.DataProductsHandler.GetDatasetsMinimal).Build(log),
		GetDataset:         transport.For(h.DataProductsHandler.GetDataset).Build(log),
		// FIXME: should perhaps not marshal the response
		CreateDataset:                      transport.For(h.DataProductsHandler.CreateDataset).RequestFromJSON().Build(log),
		UpdateDataset:                      transport.For(h.DataProductsHandler.UpdateDataset).RequestFromJSON().Build(log),
		DeleteDataset:                      transport.For(h.DataProductsHandler.DeleteDataset).Build(log),
		GetAccessiblePseudoDatasetsForUser: transport.For(h.DataProductsHandler.GetAccessiblePseudoDatasetsForUser).Build(log),

		// Metabase endpoints
		MapDataset: transport.For(h.MetabaseHandler.MapDataset).RequestFromJSON().Build(log),

		// Access endpoints
		GetAccessRequests:     transport.For(h.AccessHandler.GetAccessRequests).Build(log),
		ProcessAccessRequest:  transport.For(h.AccessHandler.ProcessAccessRequest).Build(log),
		CreateAccessRequest:   transport.For(h.AccessHandler.NewAccessRequest).RequestFromJSON().Build(log),
		DeleteAccessRequest:   transport.For(h.AccessHandler.DeleteAccessRequest).Build(log),
		UpdateAccessRequest:   transport.For(h.AccessHandler.UpdateAccessRequest).RequestFromJSON().Build(log),
		GrantAccessToDataset:  transport.For(h.AccessHandler.GrantAccessToDataset).RequestFromJSON().Build(log),
		RevokeAccessToDataset: transport.For(h.AccessHandler.RevokeAccessToDataset).Build(log),

		// Product areas endpoints
		GetProductAreas:          transport.For(h.ProductAreasHandler.GetProductAreas).Build(log),
		GetProductAreaWithAssets: transport.For(h.ProductAreasHandler.GetProductAreaWithAssets).Build(log),

		// BigQuery endpoints
		GetBigQueryColumns:  transport.For(h.BigQueryHandler.GetBigQueryColumns).Build(log),
		GetBigQueryTables:   transport.For(h.BigQueryHandler.GetBigQueryTables).Build(log),
		GetBigQueryDatasets: transport.For(h.BigQueryHandler.GetBigQueryDatasets).Build(log),
		SyncBigQueryTables:  transport.For(h.BigQueryHandler.SyncBigQueryTables).Build(log),

		// Search endpoint
		Search: transport.For(h.SearchHandler.Search).Build(log),

		// User endpoint
		GetUserData: transport.For(h.UserHandler.GetUserData).Build(log),

		// Slack endpoint
		IsValidSlackChannel: transport.For(h.SlackHandler.IsValidSlackChannel).Build(log),

		// Joinable views endpoint
		CreateJoinableViews:     transport.For(h.JoinableViewsHandler.CreateJoinableViews).RequestFromJSON().Build(log),
		GetJoinableViewsForUser: transport.For(h.JoinableViewsHandler.GetJoinableViewsForUser).Build(log),
		GetJoinableView:         transport.For(h.JoinableViewsHandler.GetJoinableView).Build(log),

		// Insight product endpoint
		GetInsightProduct:    transport.For(h.InsightProductHandler.GetInsightProduct).Build(log),
		CreateInsightProduct: transport.For(h.InsightProductHandler.CreateInsightProduct).RequestFromJSON().Build(log),
		UpdateInsightProduct: transport.For(h.InsightProductHandler.UpdateInsightProduct).RequestFromJSON().Build(log),
		DeleteInsightProduct: transport.For(h.InsightProductHandler.DeleteInsightProduct).Build(log),

		// Teamkatalogen endpoint
		SearchTeamKatalogen: transport.For(h.TeamKatalogenHandler.SearchTeamKatalogen).Build(log),

		// Polly endpoint
		SearchPolly: transport.For(h.PollyHandler.SearchPolly).Build(log),

		// Keywords endpoint
		GetKeywordsListSortedByPopularity: transport.For(h.KeywordsHandler.GetKeywordsListSortedByPopularity).Build(log),
		UpdateKeywords:                    transport.For(h.KeywordsHandler.UpdateKeywords).RequestFromJSON().Build(log),
	}
}

type Handlers struct {
	StoryHandler          *storyHandler
	TokenHandler          *tokenHandler
	DataProductsHandler   *dataProductsHandler
	MetabaseHandler       *metabaseHandler
	AccessHandler         *accessHandler
	ProductAreasHandler   *productAreasHandler
	BigQueryHandler       *bigQueryHandler
	SearchHandler         *searchHandler
	UserHandler           *userHandler
	SlackHandler          *slackHandler
	JoinableViewsHandler  *joinableViewsHandler
	InsightProductHandler *insightProductHandler
	TeamKatalogenHandler  *teamkatalogenHandler
	PollyHandler          *pollyHandler
	KeywordsHandler       *keywordsHandler
}

func NewHandlers(
	s *core.Services,
	amplitude amplitude.Amplitude,
	cfg config.Config,
) *Handlers {
	return &Handlers{
		StoryHandler:          NewStoryHandler(s.StoryService, s.TokenService, amplitude),
		TokenHandler:          NewTokenHandler(s.TokenService),
		DataProductsHandler:   NewDataProductsHandler(s.DataProductService),
		MetabaseHandler:       NewMetabaseHandler(s.MetaBaseService),
		AccessHandler:         NewAccessHandler(s.AccessService, s.MetaBaseService, cfg.Metabase.GCPProject),
		ProductAreasHandler:   NewProductAreasHandler(s.ProductAreaService),
		BigQueryHandler:       NewBigQueryHandler(s.BigQueryService),
		SearchHandler:         NewSearchHandler(s.SearchService),
		UserHandler:           NewUserHandler(s.UserService),
		SlackHandler:          NewSlackHandler(s.SlackService),
		JoinableViewsHandler:  NewJoinableViewsHandler(s.JoinableViewService),
		InsightProductHandler: NewInsightProductHandler(s.InsightProductService),
		TeamKatalogenHandler:  NewTeamKatalogenHandler(s.TeamKatalogenService),
		PollyHandler:          NewPollyHandler(s.PollyService),
		KeywordsHandler:       NewKeywordsHandler(s.KeyWordService),
	}
}
