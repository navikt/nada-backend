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
		GetStoryMetadata: HandlerFor(h.StoryHandler.GetStoryMetadata).ResponseToJSON().Build(log),
		CreateStory:      HandlerFor(h.StoryHandler.CreateStory).ResponseToJSON().Build(log),
		UpdateStory:      HandlerFor(h.StoryHandler.UpdateStory).RequestFromJSON().ResponseToJSON().Build(log),
		DeleteStory:      HandlerFor(h.StoryHandler.DeleteStory).ResponseToJSON().Build(log),

		// Token endpoints
		GetAllTeamTokens: h.TokenHandler.GetAllTeamTokens,
		RotateNadaToken:  HandlerFor(h.TokenHandler.RotateNadaToken).ResponseToJSON().Build(log),

		// Data product endpoints
		GetDataProduct:     HandlerFor(h.DataProductsHandler.GetDataProduct).ResponseToJSON().Build(log),
		CreateDataProduct:  HandlerFor(h.DataProductsHandler.CreateDataProduct).RequestFromJSON().ResponseToJSON().Build(log),
		DeleteDataProduct:  HandlerFor(h.DataProductsHandler.DeleteDataProduct).ResponseToJSON().Build(log),
		UpdateDataProduct:  HandlerFor(h.DataProductsHandler.UpdateDataProduct).RequestFromJSON().ResponseToJSON().Build(log),
		GetDatasetsMinimal: HandlerFor(h.DataProductsHandler.GetDatasetsMinimal).ResponseToJSON().Build(log),
		GetDataset:         HandlerFor(h.DataProductsHandler.GetDataset).ResponseToJSON().Build(log),
		// FIXME: should perhaps not marshal the response
		CreateDataset:                      HandlerFor(h.DataProductsHandler.CreateDataset).RequestFromJSON().ResponseToJSON().Build(log),
		UpdateDataset:                      HandlerFor(h.DataProductsHandler.UpdateDataset).RequestFromJSON().ResponseToJSON().Build(log),
		DeleteDataset:                      HandlerFor(h.DataProductsHandler.DeleteDataset).ResponseToJSON().Build(log),
		GetAccessiblePseudoDatasetsForUser: HandlerFor(h.DataProductsHandler.GetAccessiblePseudoDatasetsForUser).ResponseToJSON().Build(log),

		// Metabase endpoints
		MapDataset: HandlerFor(h.MetabaseHandler.MapDataset).RequestFromJSON().ResponseToJSON().Build(log),

		// Access endpoints
		GetAccessRequests:     HandlerFor(h.AccessHandler.GetAccessRequests).ResponseToJSON().Build(log),
		ProcessAccessRequest:  HandlerFor(h.AccessHandler.ProcessAccessRequest).ResponseToJSON().Build(log),
		CreateAccessRequest:   HandlerFor(h.AccessHandler.NewAccessRequest).RequestFromJSON().ResponseToJSON().Build(log),
		DeleteAccessRequest:   HandlerFor(h.AccessHandler.DeleteAccessRequest).ResponseToJSON().Build(log),
		UpdateAccessRequest:   HandlerFor(h.AccessHandler.UpdateAccessRequest).RequestFromJSON().ResponseToJSON().Build(log),
		GrantAccessToDataset:  HandlerFor(h.AccessHandler.GrantAccessToDataset).RequestFromJSON().ResponseToJSON().Build(log),
		RevokeAccessToDataset: HandlerFor(h.AccessHandler.RevokeAccessToDataset).ResponseToJSON().Build(log),

		// Product areas endpoints
		GetProductAreas:          HandlerFor(h.ProductAreasHandler.GetProductAreas).ResponseToJSON().Build(log),
		GetProductAreaWithAssets: HandlerFor(h.ProductAreasHandler.GetProductAreaWithAssets).ResponseToJSON().Build(log),

		// BigQuery endpoints
		GetBigQueryColumns:  HandlerFor(h.BigQueryHandler.GetBigQueryColumns).ResponseToJSON().Build(log),
		GetBigQueryTables:   HandlerFor(h.BigQueryHandler.GetBigQueryTables).ResponseToJSON().Build(log),
		GetBigQueryDatasets: HandlerFor(h.BigQueryHandler.GetBigQueryDatasets).ResponseToJSON().Build(log),
		SyncBigQueryTables:  HandlerFor(h.BigQueryHandler.SyncBigQueryTables).ResponseToJSON().Build(log),

		// Search endpoint
		Search: HandlerFor(h.SearchHandler.Search).ResponseToJSON().Build(log),

		// User endpoint
		GetUserData: HandlerFor(h.UserHandler.GetUserData).ResponseToJSON().Build(log),

		// Slack endpoint
		IsValidSlackChannel: HandlerFor(h.SlackHandler.IsValidSlackChannel).ResponseToJSON().Build(log),

		// Joinable views endpoint
		CreateJoinableViews:     HandlerFor(h.JoinableViewsHandler.CreateJoinableViews).RequestFromJSON().ResponseToJSON().Build(log),
		GetJoinableViewsForUser: HandlerFor(h.JoinableViewsHandler.GetJoinableViewsForUser).ResponseToJSON().Build(log),
		GetJoinableView:         HandlerFor(h.JoinableViewsHandler.GetJoinableView).ResponseToJSON().Build(log),

		// Insight product endpoint
		GetInsightProduct:    HandlerFor(h.InsightProductHandler.GetInsightProduct).ResponseToJSON().Build(log),
		CreateInsightProduct: HandlerFor(h.InsightProductHandler.CreateInsightProduct).RequestFromJSON().ResponseToJSON().Build(log),
		UpdateInsightProduct: HandlerFor(h.InsightProductHandler.UpdateInsightProduct).RequestFromJSON().ResponseToJSON().Build(log),
		DeleteInsightProduct: HandlerFor(h.InsightProductHandler.DeleteInsightProduct).ResponseToJSON().Build(log),

		// Teamkatalogen endpoint
		SearchTeamKatalogen: HandlerFor(h.TeamKatalogenHandler.SearchTeamKatalogen).ResponseToJSON().Build(log),

		// Polly endpoint
		SearchPolly: HandlerFor(h.PollyHandler.SearchPolly).ResponseToJSON().Build(log),

		// Keywords endpoint
		GetKeywordsListSortedByPopularity: HandlerFor(h.KeywordsHandler.GetKeywordsListSortedByPopularity).ResponseToJSON().Build(log),
		UpdateKeywords:                    HandlerFor(h.KeywordsHandler.UpdateKeywords).RequestFromJSON().ResponseToJSON().Build(log),
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
