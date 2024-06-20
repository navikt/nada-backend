package handlers

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/config/v2"
	"github.com/navikt/nada-backend/pkg/service/core"
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
		GetStoryMetadata: TransportFor(h.StoryHandler.GetStoryMetadata).Build(log),
		CreateStory:      TransportFor(h.StoryHandler.CreateStory).Build(log),
		UpdateStory:      TransportFor(h.StoryHandler.UpdateStory).RequestFromJSON().Build(log),
		DeleteStory:      TransportFor(h.StoryHandler.DeleteStory).Build(log),

		// Token endpoints
		GetAllTeamTokens: h.TokenHandler.GetAllTeamTokens,
		RotateNadaToken:  TransportFor(h.TokenHandler.RotateNadaToken).Build(log),

		// Data product endpoints
		GetDataProduct:     TransportFor(h.DataProductsHandler.GetDataProduct).Build(log),
		CreateDataProduct:  TransportFor(h.DataProductsHandler.CreateDataProduct).RequestFromJSON().Build(log),
		DeleteDataProduct:  TransportFor(h.DataProductsHandler.DeleteDataProduct).Build(log),
		UpdateDataProduct:  TransportFor(h.DataProductsHandler.UpdateDataProduct).RequestFromJSON().Build(log),
		GetDatasetsMinimal: TransportFor(h.DataProductsHandler.GetDatasetsMinimal).Build(log),
		GetDataset:         TransportFor(h.DataProductsHandler.GetDataset).Build(log),
		// FIXME: should perhaps not marshal the response
		CreateDataset:                      TransportFor(h.DataProductsHandler.CreateDataset).RequestFromJSON().Build(log),
		UpdateDataset:                      TransportFor(h.DataProductsHandler.UpdateDataset).RequestFromJSON().Build(log),
		DeleteDataset:                      TransportFor(h.DataProductsHandler.DeleteDataset).Build(log),
		GetAccessiblePseudoDatasetsForUser: TransportFor(h.DataProductsHandler.GetAccessiblePseudoDatasetsForUser).Build(log),

		// Metabase endpoints
		MapDataset: TransportFor(h.MetabaseHandler.MapDataset).RequestFromJSON().Build(log),

		// Access endpoints
		GetAccessRequests:     TransportFor(h.AccessHandler.GetAccessRequests).Build(log),
		ProcessAccessRequest:  TransportFor(h.AccessHandler.ProcessAccessRequest).Build(log),
		CreateAccessRequest:   TransportFor(h.AccessHandler.NewAccessRequest).RequestFromJSON().Build(log),
		DeleteAccessRequest:   TransportFor(h.AccessHandler.DeleteAccessRequest).Build(log),
		UpdateAccessRequest:   TransportFor(h.AccessHandler.UpdateAccessRequest).RequestFromJSON().Build(log),
		GrantAccessToDataset:  TransportFor(h.AccessHandler.GrantAccessToDataset).RequestFromJSON().Build(log),
		RevokeAccessToDataset: TransportFor(h.AccessHandler.RevokeAccessToDataset).Build(log),

		// Product areas endpoints
		GetProductAreas:          TransportFor(h.ProductAreasHandler.GetProductAreas).Build(log),
		GetProductAreaWithAssets: TransportFor(h.ProductAreasHandler.GetProductAreaWithAssets).Build(log),

		// BigQuery endpoints
		GetBigQueryColumns:  TransportFor(h.BigQueryHandler.GetBigQueryColumns).Build(log),
		GetBigQueryTables:   TransportFor(h.BigQueryHandler.GetBigQueryTables).Build(log),
		GetBigQueryDatasets: TransportFor(h.BigQueryHandler.GetBigQueryDatasets).Build(log),
		SyncBigQueryTables:  TransportFor(h.BigQueryHandler.SyncBigQueryTables).Build(log),

		// Search endpoint
		Search: TransportFor(h.SearchHandler.Search).Build(log),

		// User endpoint
		GetUserData: TransportFor(h.UserHandler.GetUserData).Build(log),

		// Slack endpoint
		IsValidSlackChannel: TransportFor(h.SlackHandler.IsValidSlackChannel).Build(log),

		// Joinable views endpoint
		CreateJoinableViews:     TransportFor(h.JoinableViewsHandler.CreateJoinableViews).RequestFromJSON().Build(log),
		GetJoinableViewsForUser: TransportFor(h.JoinableViewsHandler.GetJoinableViewsForUser).Build(log),
		GetJoinableView:         TransportFor(h.JoinableViewsHandler.GetJoinableView).Build(log),

		// Insight product endpoint
		GetInsightProduct:    TransportFor(h.InsightProductHandler.GetInsightProduct).Build(log),
		CreateInsightProduct: TransportFor(h.InsightProductHandler.CreateInsightProduct).RequestFromJSON().Build(log),
		UpdateInsightProduct: TransportFor(h.InsightProductHandler.UpdateInsightProduct).RequestFromJSON().Build(log),
		DeleteInsightProduct: TransportFor(h.InsightProductHandler.DeleteInsightProduct).Build(log),

		// Teamkatalogen endpoint
		SearchTeamKatalogen: TransportFor(h.TeamKatalogenHandler.SearchTeamKatalogen).Build(log),

		// Polly endpoint
		SearchPolly: TransportFor(h.PollyHandler.SearchPolly).Build(log),

		// Keywords endpoint
		GetKeywordsListSortedByPopularity: TransportFor(h.KeywordsHandler.GetKeywordsListSortedByPopularity).Build(log),
		UpdateKeywords:                    TransportFor(h.KeywordsHandler.UpdateKeywords).RequestFromJSON().Build(log),
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
