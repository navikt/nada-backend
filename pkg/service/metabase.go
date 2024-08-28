package service

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	MetabaseRestrictedCollectionTag = "🔐"
)

type MetabaseStorage interface {
	CreateMetadata(ctx context.Context, datasetID uuid.UUID) error
	DeleteMetadata(ctx context.Context, datasetID uuid.UUID) error
	DeleteRestrictedMetadata(ctx context.Context, datasetID uuid.UUID) error
	GetAllMetadata(ctx context.Context) ([]*MetabaseMetadata, error)
	GetMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*MetabaseMetadata, error)
	GetOpenTablesInSameBigQueryDataset(ctx context.Context, projectID, dataset string) ([]string, error)
	RestoreMetadata(ctx context.Context, datasetID uuid.UUID) error
	SetCollectionMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, collectionID int) (*MetabaseMetadata, error)
	SetDatabaseMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, databaseID int) (*MetabaseMetadata, error)
	SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) (*MetabaseMetadata, error)
	SetServiceAccountMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, saEmail string) (*MetabaseMetadata, error)
	SetSyncCompletedMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error
	SoftDeleteMetadata(ctx context.Context, datasetID uuid.UUID) error
}

type MetabaseAPI interface {
	AddPermissionGroupMember(ctx context.Context, groupID int, email string) error
	ArchiveCollection(ctx context.Context, colID int) error
	AutoMapSemanticTypes(ctx context.Context, dbID int) error
	CreateCollection(ctx context.Context, name string) (int, error)
	CreateCollectionWithAccess(ctx context.Context, groupID int, name string) (int, error)
	CreateDatabase(ctx context.Context, team, name, saJSON, saEmail string, ds *BigQuery) (int, error)
	GetPermissionGroups(ctx context.Context) ([]MetabasePermissionGroup, error)
	GetOrCreatePermissionGroup(ctx context.Context, name string) (int, error)
	CreatePermissionGroup(ctx context.Context, name string) (int, error)
	Databases(ctx context.Context) ([]MetabaseDatabase, error)
	DeleteDatabase(ctx context.Context, id int) error
	DeletePermissionGroup(ctx context.Context, groupID int) error
	EnsureValidSession(ctx context.Context) error
	GetPermissionGroup(ctx context.Context, groupID int) ([]MetabasePermissionGroupMember, error)
	HideTables(ctx context.Context, ids []int) error
	OpenAccessToDatabase(ctx context.Context, databaseID int) error
	PerformRequest(ctx context.Context, method, path string, buffer io.ReadWriter) (*http.Response, error)
	RemovePermissionGroupMember(ctx context.Context, memberID int) error
	RestrictAccessToDatabase(ctx context.Context, groupID int, databaseID int) error
	SetCollectionAccess(ctx context.Context, groupID int, collectionID int) error
	ShowTables(ctx context.Context, ids []int) error
	Tables(ctx context.Context, dbID int) ([]MetabaseTable, error)
	GetCollections(ctx context.Context) ([]*MetabaseCollection, error)
	UpdateCollection(ctx context.Context, collection *MetabaseCollection) error
}

type MetabaseService interface {
	SyncTableVisibility(ctx context.Context, mbMeta *MetabaseMetadata, bq BigQuery) error
	SyncAllTablesVisibility(ctx context.Context) error
	RevokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error
	RevokeMetabaseAccessFromAccessID(ctx context.Context, accessID uuid.UUID) error
	DeleteDatabase(ctx context.Context, dsID uuid.UUID) error
	GrantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject, subjectType string) error
	CreateMappingRequest(ctx context.Context, user *User, datasetID uuid.UUID, services []string) error
	MapDataset(ctx context.Context, datasetID uuid.UUID, services []string) error
}

type MetabaseField struct{}

type MetabaseTable struct {
	Name   string `json:"name"`
	ID     int    `json:"id"`
	Fields []struct {
		DatabaseType string `json:"database_type"`
		ID           int    `json:"id"`
		SemanticType string `json:"semantic_type"`
	} `json:"fields"`
}

type MetabasePermissionGroup struct {
	ID      int                             `json:"id"`
	Name    string                          `json:"name"`
	Members []MetabasePermissionGroupMember `json:"members"`
}

type MetabasePermissionGroupMember struct {
	ID    int    `json:"membership_id"`
	Email string `json:"email"`
}

type MetabaseUser struct {
	Email string `json:"email"`
	ID    int    `json:"id"`
}

type MetabaseDatabase struct {
	ID        int
	Name      string
	DatasetID string
	ProjectID string
	NadaID    string
	SAEmail   string
}

type MetabaseMetadata struct {
	DatasetID         uuid.UUID
	DatabaseID        *int
	PermissionGroupID *int
	CollectionID      *int
	SAEmail           string
	DeletedAt         *time.Time
	SyncCompleted     *time.Time
}

// MetabaseCollection represents a subset of the metadata returned
// for a Metabase collection
type MetabaseCollection struct {
	ID          int
	Name        string
	Description string
}
