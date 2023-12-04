// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.23.0

package gensql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Querier interface {
	AddTeamProject(ctx context.Context, arg AddTeamProjectParams) (TeamProject, error)
	ApproveAccessRequest(ctx context.Context, arg ApproveAccessRequestParams) error
	CleanupStoryDrafts(ctx context.Context) error
	ClearTeamProjectsCache(ctx context.Context) error
	CreateAccessRequestForDataset(ctx context.Context, arg CreateAccessRequestForDatasetParams) (DatasetAccessRequest, error)
	CreateBigqueryDatasource(ctx context.Context, arg CreateBigqueryDatasourceParams) (DatasourceBigquery, error)
	CreateDataproduct(ctx context.Context, arg CreateDataproductParams) (Dataproduct, error)
	CreateDataset(ctx context.Context, arg CreateDatasetParams) (Dataset, error)
	CreateInsightProduct(ctx context.Context, arg CreateInsightProductParams) (InsightProduct, error)
	CreateJoinableViews(ctx context.Context, arg CreateJoinableViewsParams) (JoinableView, error)
	CreateJoinableViewsDatasource(ctx context.Context, arg CreateJoinableViewsDatasourceParams) (JoinableViewsDatasource, error)
	CreateMetabaseMetadata(ctx context.Context, arg CreateMetabaseMetadataParams) error
	CreatePollyDocumentation(ctx context.Context, arg CreatePollyDocumentationParams) (PollyDocumentation, error)
	CreateQuartoStory(ctx context.Context, arg CreateQuartoStoryParams) (QuartoStory, error)
	CreateQuartoStoryWithID(ctx context.Context, arg CreateQuartoStoryWithIDParams) (QuartoStory, error)
	CreateSession(ctx context.Context, arg CreateSessionParams) error
	CreateStory(ctx context.Context, arg CreateStoryParams) (Story, error)
	CreateStoryDraft(ctx context.Context, name string) (StoryDraft, error)
	CreateStoryView(ctx context.Context, arg CreateStoryViewParams) (StoryView, error)
	CreateStoryViewDraft(ctx context.Context, arg CreateStoryViewDraftParams) (StoryViewDraft, error)
	CreateTagIfNotExist(ctx context.Context, phrase string) error
	CreateTeamAndProductAreaMapping(ctx context.Context, arg CreateTeamAndProductAreaMappingParams) (TeamProductareaMapping, error)
	DataproductGroupStats(ctx context.Context, arg DataproductGroupStatsParams) ([]DataproductGroupStatsRow, error)
	DataproductKeywords(ctx context.Context, keyword string) ([]DataproductKeywordsRow, error)
	DatasetsByMetabase(ctx context.Context, arg DatasetsByMetabaseParams) ([]Dataset, error)
	DeleteAccessRequest(ctx context.Context, id uuid.UUID) error
	DeleteDataproduct(ctx context.Context, id uuid.UUID) error
	DeleteDataset(ctx context.Context, id uuid.UUID) error
	DeleteInsightProduct(ctx context.Context, id uuid.UUID) error
	DeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error
	DeleteNadaToken(ctx context.Context, team string) error
	DeleteQuartoStory(ctx context.Context, id uuid.UUID) error
	DeleteSession(ctx context.Context, token string) error
	DeleteStory(ctx context.Context, id uuid.UUID) error
	DeleteStoryDraft(ctx context.Context, id uuid.UUID) error
	DeleteStoryViewDraft(ctx context.Context, storyID uuid.UUID) error
	DeleteStoryViews(ctx context.Context, storyID uuid.UUID) error
	DenyAccessRequest(ctx context.Context, arg DenyAccessRequestParams) error
	GetAccessRequest(ctx context.Context, id uuid.UUID) (DatasetAccessRequest, error)
	GetAccessToDataset(ctx context.Context, id uuid.UUID) (DatasetAccess, error)
	GetAccessiblePseudoDatasetsByUser(ctx context.Context, arg GetAccessiblePseudoDatasetsByUserParams) ([]GetAccessiblePseudoDatasetsByUserRow, error)
	GetActiveAccessToDatasetForSubject(ctx context.Context, arg GetActiveAccessToDatasetForSubjectParams) (DatasetAccess, error)
	GetAllMetabaseMetadata(ctx context.Context) ([]MetabaseMetadatum, error)
	GetBigqueryDatasource(ctx context.Context, arg GetBigqueryDatasourceParams) (DatasourceBigquery, error)
	GetBigqueryDatasources(ctx context.Context) ([]DatasourceBigquery, error)
	GetDashboard(ctx context.Context, id string) (Dashboard, error)
	GetDataproduct(ctx context.Context, id uuid.UUID) (Dataproduct, error)
	GetDataproducts(ctx context.Context, arg GetDataproductsParams) ([]Dataproduct, error)
	GetDataproductsByGroups(ctx context.Context, groups []string) ([]Dataproduct, error)
	GetDataproductsByIDs(ctx context.Context, ids []uuid.UUID) ([]Dataproduct, error)
	GetDataproductsByProductArea(ctx context.Context, productAreaID sql.NullString) ([]Dataproduct, error)
	GetDataproductsByTeam(ctx context.Context, teamID sql.NullString) ([]Dataproduct, error)
	GetDataset(ctx context.Context, id uuid.UUID) (Dataset, error)
	GetDatasetMappings(ctx context.Context, datasetID uuid.UUID) (ThirdPartyMapping, error)
	GetDatasets(ctx context.Context, arg GetDatasetsParams) ([]Dataset, error)
	GetDatasetsByGroups(ctx context.Context, groups []string) ([]Dataset, error)
	GetDatasetsByIDs(ctx context.Context, ids []uuid.UUID) ([]Dataset, error)
	GetDatasetsByMapping(ctx context.Context, arg GetDatasetsByMappingParams) ([]Dataset, error)
	GetDatasetsByUserAccess(ctx context.Context, id string) ([]Dataset, error)
	GetDatasetsForOwner(ctx context.Context, groups []string) ([]Dataset, error)
	GetDatasetsInDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]Dataset, error)
	GetInsightProduct(ctx context.Context, id uuid.UUID) (InsightProduct, error)
	GetInsightProductByGroups(ctx context.Context, groups []string) ([]InsightProduct, error)
	GetInsightProducts(ctx context.Context) ([]InsightProduct, error)
	GetInsightProductsByIDs(ctx context.Context, ids []uuid.UUID) ([]InsightProduct, error)
	GetInsightProductsByProductArea(ctx context.Context, productAreaID sql.NullString) ([]InsightProduct, error)
	GetInsightProductsByTeam(ctx context.Context, teamID sql.NullString) ([]InsightProduct, error)
	GetJoinableViewWithDataset(ctx context.Context, id uuid.UUID) ([]GetJoinableViewWithDatasetRow, error)
	GetJoinableViewsForOwner(ctx context.Context, owner string) ([]GetJoinableViewsForOwnerRow, error)
	GetJoinableViewsForReferenceAndUser(ctx context.Context, arg GetJoinableViewsForReferenceAndUserParams) ([]GetJoinableViewsForReferenceAndUserRow, error)
	GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]GetJoinableViewsToBeDeletedWithRefDatasourceRow, error)
	GetJoinableViewsWithReference(ctx context.Context) ([]GetJoinableViewsWithReferenceRow, error)
	GetKeywords(ctx context.Context) ([]GetKeywordsRow, error)
	GetMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) (MetabaseMetadatum, error)
	GetMetabaseMetadataWithDeleted(ctx context.Context, datasetID uuid.UUID) (MetabaseMetadatum, error)
	GetNadaToken(ctx context.Context, team string) (uuid.UUID, error)
	GetNadaTokens(ctx context.Context, teams []string) ([]NadaToken, error)
	GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error)
	GetPollyDocumentation(ctx context.Context, id uuid.UUID) (PollyDocumentation, error)
	GetPseudoDatasourcesToDelete(ctx context.Context) ([]DatasourceBigquery, error)
	GetQuartoStories(ctx context.Context) ([]QuartoStory, error)
	GetQuartoStoriesByGroups(ctx context.Context, groups []string) ([]QuartoStory, error)
	GetQuartoStoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]QuartoStory, error)
	GetQuartoStoriesByProductArea(ctx context.Context, productAreaID sql.NullString) ([]QuartoStory, error)
	GetQuartoStoriesByTeam(ctx context.Context, teamID sql.NullString) ([]QuartoStory, error)
	GetQuartoStory(ctx context.Context, id uuid.UUID) (QuartoStory, error)
	GetSession(ctx context.Context, token string) (Session, error)
	GetStories(ctx context.Context) ([]Story, error)
	GetStoriesByGroups(ctx context.Context, groups []string) ([]Story, error)
	GetStoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]Story, error)
	GetStoriesByProductArea(ctx context.Context, productAreaID sql.NullString) ([]Story, error)
	GetStoriesByTeam(ctx context.Context, teamID sql.NullString) ([]Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (Story, error)
	GetStoryDraft(ctx context.Context, id uuid.UUID) (StoryDraft, error)
	GetStoryDrafts(ctx context.Context) ([]StoryDraft, error)
	GetStoryFromToken(ctx context.Context, token uuid.UUID) (Story, error)
	GetStoryToken(ctx context.Context, storyID uuid.UUID) (StoryToken, error)
	GetStoryView(ctx context.Context, id uuid.UUID) (StoryView, error)
	GetStoryViewDraft(ctx context.Context, id uuid.UUID) (StoryViewDraft, error)
	GetStoryViewDrafts(ctx context.Context, storyID uuid.UUID) ([]StoryViewDraft, error)
	GetStoryViews(ctx context.Context, storyID uuid.UUID) ([]StoryView, error)
	GetTag(ctx context.Context) (Tag, error)
	GetTagByPhrase(ctx context.Context) (Tag, error)
	GetTags(ctx context.Context) ([]Tag, error)
	GetTeamAndProductAreaID(ctx context.Context, teamID string) (TeamProductareaMapping, error)
	GetTeamFromNadaToken(ctx context.Context, token uuid.UUID) (string, error)
	GetTeamProjects(ctx context.Context) ([]TeamProject, error)
	GetTeamsAndProductAreaIDs(ctx context.Context) ([]TeamProductareaMapping, error)
	GrantAccessToDataset(ctx context.Context, arg GrantAccessToDatasetParams) (DatasetAccess, error)
	ListAccessRequestsForDataset(ctx context.Context, datasetID uuid.UUID) ([]DatasetAccessRequest, error)
	ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]DatasetAccessRequest, error)
	ListAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]DatasetAccess, error)
	ListActiveAccessToDataset(ctx context.Context, datasetID uuid.UUID) ([]DatasetAccess, error)
	ListUnrevokedExpiredAccessEntries(ctx context.Context) ([]DatasetAccess, error)
	MapDataset(ctx context.Context, arg MapDatasetParams) error
	RemoveKeywordInDatasets(ctx context.Context, keywordToRemove interface{}) error
	RemoveKeywordInStories(ctx context.Context, keywordToRemove interface{}) error
	ReplaceDatasetsTag(ctx context.Context, arg ReplaceDatasetsTagParams) error
	ReplaceKeywordInDatasets(ctx context.Context, arg ReplaceKeywordInDatasetsParams) error
	ReplaceKeywordInStories(ctx context.Context, arg ReplaceKeywordInStoriesParams) error
	ReplaceQuartoStoriesTag(ctx context.Context, arg ReplaceQuartoStoriesTagParams) error
	ReplaceStoriesTag(ctx context.Context, arg ReplaceStoriesTagParams) error
	RestoreMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error
	RevokeAccessToDataset(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, arg SearchParams) ([]SearchRow, error)
	SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error
	SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error
	SetPermissionGroupMetabaseMetadata(ctx context.Context, arg SetPermissionGroupMetabaseMetadataParams) error
	SoftDeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error
	UpdateAccessRequest(ctx context.Context, arg UpdateAccessRequestParams) (DatasetAccessRequest, error)
	UpdateBigqueryDatasource(ctx context.Context, arg UpdateBigqueryDatasourceParams) error
	UpdateBigqueryDatasourceMissing(ctx context.Context, datasetID uuid.UUID) error
	UpdateBigqueryDatasourceSchema(ctx context.Context, arg UpdateBigqueryDatasourceSchemaParams) error
	UpdateDataproduct(ctx context.Context, arg UpdateDataproductParams) (Dataproduct, error)
	UpdateDataset(ctx context.Context, arg UpdateDatasetParams) (Dataset, error)
	UpdateInsightProduct(ctx context.Context, arg UpdateInsightProductParams) (InsightProduct, error)
	UpdateProductAreaForTeam(ctx context.Context, arg UpdateProductAreaForTeamParams) error
	UpdateQuartoStory(ctx context.Context, arg UpdateQuartoStoryParams) (QuartoStory, error)
	UpdateStory(ctx context.Context, arg UpdateStoryParams) (Story, error)
	UpdateTag(ctx context.Context, arg UpdateTagParams) error
}

var _ Querier = (*Queries)(nil)
