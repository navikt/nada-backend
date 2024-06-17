package handlers

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/service/core"
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

func NewEndpoints(h *Handlers) *Endpoints {
	return &Endpoints{
		// Story endpoints
		GetGCSObject:     h.StoryHandler.GetGCSObject,
		CreateStoryHTTP:  h.StoryHandler.CreateStoryHTTP,
		UpdateStoryHTTP:  h.StoryHandler.UpdateStoryHTTP,
		AppendStoryHTTP:  h.StoryHandler.AppendStoryHTTP,
		GetStoryMetadata: HandlerFor(h.StoryHandler.GetStoryMetadata).ResponseToJSON().Build(),
		CreateStory:      HandlerFor(h.StoryHandler.CreateStory).ResponseToJSON().Build(),
		UpdateStory:      HandlerFor(h.StoryHandler.UpdateStory).RequestFromJSON().ResponseToJSON().Build(),
		DeleteStory:      HandlerFor(h.StoryHandler.DeleteStory).ResponseToJSON().Build(),

		// Token endpoints
		GetAllTeamTokens: h.TokenHandler.GetAllTeamTokens,
		RotateNadaToken:  HandlerFor(h.TokenHandler.RotateNadaToken).ResponseToJSON().Build(),

		// Data product endpoints
		GetDataProduct:     HandlerFor(h.DataProductsHandler.GetDataProduct).ResponseToJSON().Build(),
		CreateDataProduct:  HandlerFor(h.DataProductsHandler.CreateDataProduct).RequestFromJSON().ResponseToJSON().Build(),
		DeleteDataProduct:  HandlerFor(h.DataProductsHandler.DeleteDataProduct).ResponseToJSON().Build(),
		UpdateDataProduct:  HandlerFor(h.DataProductsHandler.UpdateDataProduct).RequestFromJSON().ResponseToJSON().Build(),
		GetDatasetsMinimal: HandlerFor(h.DataProductsHandler.GetDatasetsMinimal).ResponseToJSON().Build(),
		GetDataset:         HandlerFor(h.DataProductsHandler.GetDataset).ResponseToJSON().Build(),
		// FIXME: should perhaps not marshal the response
		CreateDataset:                      HandlerFor(h.DataProductsHandler.CreateDataset).RequestFromJSON().ResponseToJSON().Build(),
		UpdateDataset:                      HandlerFor(h.DataProductsHandler.UpdateDataset).RequestFromJSON().ResponseToJSON().Build(),
		DeleteDataset:                      HandlerFor(h.DataProductsHandler.DeleteDataset).ResponseToJSON().Build(),
		GetAccessiblePseudoDatasetsForUser: HandlerFor(h.DataProductsHandler.GetAccessiblePseudoDatasetsForUser).ResponseToJSON().Build(),

		// Metabase endpoints
		MapDataset: HandlerFor(h.MetabaseHandler.MapDataset).RequestFromJSON().ResponseToJSON().Build(),

		// Access endpoints
		GetAccessRequests:     HandlerFor(h.AccessHandler.GetAccessRequests).ResponseToJSON().Build(),
		ProcessAccessRequest:  HandlerFor(h.AccessHandler.ProcessAccessRequest).ResponseToJSON().Build(),
		CreateAccessRequest:   HandlerFor(h.AccessHandler.NewAccessRequest).RequestFromJSON().ResponseToJSON().Build(),
		DeleteAccessRequest:   HandlerFor(h.AccessHandler.DeleteAccessRequest).ResponseToJSON().Build(),
		UpdateAccessRequest:   HandlerFor(h.AccessHandler.UpdateAccessRequest).RequestFromJSON().ResponseToJSON().Build(),
		GrantAccessToDataset:  HandlerFor(h.AccessHandler.GrantAccessToDataset).RequestFromJSON().ResponseToJSON().Build(),
		RevokeAccessToDataset: HandlerFor(h.AccessHandler.RevokeAccessToDataset).ResponseToJSON().Build(),

		// Product areas endpoints
		GetProductAreas:          HandlerFor(h.ProductAreasHandler.GetProductAreas).ResponseToJSON().Build(),
		GetProductAreaWithAssets: HandlerFor(h.ProductAreasHandler.GetProductAreaWithAssets).ResponseToJSON().Build(),

		// BigQuery endpoints
		GetBigQueryColumns:  HandlerFor(h.BigQueryHandler.GetBigQueryColumns).ResponseToJSON().Build(),
		GetBigQueryTables:   HandlerFor(h.BigQueryHandler.GetBigQueryTables).ResponseToJSON().Build(),
		GetBigQueryDatasets: HandlerFor(h.BigQueryHandler.GetBigQueryDatasets).ResponseToJSON().Build(),
		SyncBigQueryTables:  HandlerFor(h.BigQueryHandler.SyncBigQueryTables).ResponseToJSON().Build(),

		// Search endpoint
		Search: HandlerFor(h.SearchHandler.Search).ResponseToJSON().Build(),

		// User endpoint
		GetUserData: HandlerFor(h.UserHandler.GetUserData).ResponseToJSON().Build(),

		// Slack endpoint
		IsValidSlackChannel: HandlerFor(h.SlackHandler.IsValidSlackChannel).ResponseToJSON().Build(),

		// Joinable views endpoint
		CreateJoinableViews:     HandlerFor(h.JoinableViewsHandler.CreateJoinableViews).RequestFromJSON().ResponseToJSON().Build(),
		GetJoinableViewsForUser: HandlerFor(h.JoinableViewsHandler.GetJoinableViewsForUser).ResponseToJSON().Build(),
		GetJoinableView:         HandlerFor(h.JoinableViewsHandler.GetJoinableView).ResponseToJSON().Build(),

		// Insight product endpoint
		GetInsightProduct:    HandlerFor(h.InsightProductHandler.GetInsightProduct).ResponseToJSON().Build(),
		CreateInsightProduct: HandlerFor(h.InsightProductHandler.CreateInsightProduct).RequestFromJSON().ResponseToJSON().Build(),
		UpdateInsightProduct: HandlerFor(h.InsightProductHandler.UpdateInsightProduct).RequestFromJSON().ResponseToJSON().Build(),
		DeleteInsightProduct: HandlerFor(h.InsightProductHandler.DeleteInsightProduct).ResponseToJSON().Build(),

		// Teamkatalogen endpoint
		SearchTeamKatalogen: HandlerFor(h.TeamKatalogenHandler.SearchTeamKatalogen).ResponseToJSON().Build(),

		// Polly endpoint
		SearchPolly: HandlerFor(h.PollyHandler.SearchPolly).ResponseToJSON().Build(),

		// Keywords endpoint
		GetKeywordsListSortedByPopularity: HandlerFor(h.KeywordsHandler.GetKeywordsListSortedByPopularity).ResponseToJSON().Build(),
		UpdateKeywords:                    HandlerFor(h.KeywordsHandler.UpdateKeywords).RequestFromJSON().ResponseToJSON().Build(),
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

func NewHandlers(s *core.Services, amplitude amplitude.Amplitude) *Handlers {
	return &Handlers{
		StoryHandler:          NewStoryHandler(s.StoryService, s.TokenService, amplitude),
		TokenHandler:          NewTokenHandler(s.TokenService),
		DataProductsHandler:   NewDataProductsHandler(s.DataProductService),
		MetabaseHandler:       NewMetabaseHandler(s.MetaBaseService),
		AccessHandler:         NewAccessHandler(s.AccessService),
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
