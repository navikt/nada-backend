// Code generated by sqlc. DO NOT EDIT.

package gensql

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	ApproveAccessRequest(ctx context.Context, arg ApproveAccessRequestParams) error
	CleanupStoryDrafts(ctx context.Context) error
	CreateAccessRequestForDataproduct(ctx context.Context, arg CreateAccessRequestForDataproductParams) (DataproductAccessRequest, error)
	CreateBigqueryDatasource(ctx context.Context, arg CreateBigqueryDatasourceParams) (DatasourceBigquery, error)
	CreateDataproduct(ctx context.Context, arg CreateDataproductParams) (Dataproduct, error)
	CreateDataproductRequester(ctx context.Context, arg CreateDataproductRequesterParams) error
	CreateMetabaseMetadata(ctx context.Context, arg CreateMetabaseMetadataParams) error
	CreatePollyDocumentation(ctx context.Context, arg CreatePollyDocumentationParams) (PollyDocumentation, error)
	CreateSession(ctx context.Context, arg CreateSessionParams) error
	CreateStory(ctx context.Context, arg CreateStoryParams) (Story, error)
	CreateStoryDraft(ctx context.Context, name string) (StoryDraft, error)
	CreateStoryView(ctx context.Context, arg CreateStoryViewParams) (StoryView, error)
	CreateStoryViewDraft(ctx context.Context, arg CreateStoryViewDraftParams) (StoryViewDraft, error)
	DataproductGroupStats(ctx context.Context, arg DataproductGroupStatsParams) ([]DataproductGroupStatsRow, error)
	DataproductKeywords(ctx context.Context, keyword string) ([]DataproductKeywordsRow, error)
	DataproductsByMetabase(ctx context.Context, arg DataproductsByMetabaseParams) ([]Dataproduct, error)
	DeleteAccessRequest(ctx context.Context, id uuid.UUID) error
	DeleteDataproduct(ctx context.Context, id uuid.UUID) error
	DeleteDataproductRequester(ctx context.Context, arg DeleteDataproductRequesterParams) error
	DeleteMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error
	DeleteSession(ctx context.Context, token string) error
	DeleteStory(ctx context.Context, id uuid.UUID) error
	DeleteStoryDraft(ctx context.Context, id uuid.UUID) error
	DeleteStoryViewDraft(ctx context.Context, storyID uuid.UUID) error
	DeleteStoryViews(ctx context.Context, storyID uuid.UUID) error
	DenyAccessRequest(ctx context.Context, arg DenyAccessRequestParams) error
	GetAccessRequest(ctx context.Context, id uuid.UUID) (DataproductAccessRequest, error)
	GetAccessToDataproduct(ctx context.Context, id uuid.UUID) (DataproductAccess, error)
	GetActiveAccessToDataproductForSubject(ctx context.Context, arg GetActiveAccessToDataproductForSubjectParams) (DataproductAccess, error)
	GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (DatasourceBigquery, error)
	GetBigqueryDatasources(ctx context.Context) ([]DatasourceBigquery, error)
	GetDataproduct(ctx context.Context, id uuid.UUID) (Dataproduct, error)
	GetDataproductMappings(ctx context.Context, dataproductID uuid.UUID) (ThirdPartyMapping, error)
	GetDataproductRequesters(ctx context.Context, dataproductID uuid.UUID) ([]string, error)
	GetDataproducts(ctx context.Context, arg GetDataproductsParams) ([]Dataproduct, error)
	GetDataproductsByGroups(ctx context.Context, groups []string) ([]Dataproduct, error)
	GetDataproductsByIDs(ctx context.Context, ids []uuid.UUID) ([]Dataproduct, error)
	GetDataproductsByMapping(ctx context.Context, arg GetDataproductsByMappingParams) ([]Dataproduct, error)
	GetDataproductsByUserAccess(ctx context.Context, id string) ([]Dataproduct, error)
	GetMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) (MetabaseMetadatum, error)
	GetMetabaseMetadataWithDeleted(ctx context.Context, dataproductID uuid.UUID) (MetabaseMetadatum, error)
	GetPollyDocumentation(ctx context.Context, id uuid.UUID) (PollyDocumentation, error)
	GetSession(ctx context.Context, token string) (Session, error)
	GetStories(ctx context.Context) ([]Story, error)
	GetStoriesByGroups(ctx context.Context, groups []string) ([]Story, error)
	GetStoriesByIDs(ctx context.Context, ids []uuid.UUID) ([]Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (Story, error)
	GetStoryDraft(ctx context.Context, id uuid.UUID) (StoryDraft, error)
	GetStoryDrafts(ctx context.Context) ([]StoryDraft, error)
	GetStoryFromToken(ctx context.Context, token uuid.UUID) (Story, error)
	GetStoryToken(ctx context.Context, storyID uuid.UUID) (StoryToken, error)
	GetStoryView(ctx context.Context, id uuid.UUID) (StoryView, error)
	GetStoryViewDraft(ctx context.Context, id uuid.UUID) (StoryViewDraft, error)
	GetStoryViewDrafts(ctx context.Context, storyID uuid.UUID) ([]StoryViewDraft, error)
	GetStoryViews(ctx context.Context, storyID uuid.UUID) ([]StoryView, error)
	GrantAccessToDataproduct(ctx context.Context, arg GrantAccessToDataproductParams) (DataproductAccess, error)
	ListAccessRequestsForDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]DataproductAccessRequest, error)
	ListAccessRequestsForOwner(ctx context.Context, owner []string) ([]DataproductAccessRequest, error)
	ListAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]DataproductAccess, error)
	ListActiveAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]DataproductAccess, error)
	ListUnrevokedExpiredAccessEntries(ctx context.Context) ([]DataproductAccess, error)
	MapDataproduct(ctx context.Context, arg MapDataproductParams) error
	RestoreMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error
	RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, arg SearchParams) ([]SearchRow, error)
	SetPermissionGroupMetabaseMetadata(ctx context.Context, arg SetPermissionGroupMetabaseMetadataParams) error
	SoftDeleteMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error
	UpdateAccessRequest(ctx context.Context, arg UpdateAccessRequestParams) (DataproductAccessRequest, error)
	UpdateBigqueryDatasourceSchema(ctx context.Context, arg UpdateBigqueryDatasourceSchemaParams) error
	UpdateDataproduct(ctx context.Context, arg UpdateDataproductParams) (Dataproduct, error)
	UpdateStory(ctx context.Context, arg UpdateStoryParams) (Story, error)
}

var _ Querier = (*Queries)(nil)
