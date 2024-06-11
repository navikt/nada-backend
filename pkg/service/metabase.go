package service

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type MetabaseStorage interface {
	CreateMetadata(ctx context.Context, metadata *MetabaseMetadata) error
	GetMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*MetabaseMetadata, error)
	GetAllMetadata(ctx context.Context) ([]*MetabaseMetadata, error)
	GetOpenTablesInSameBigQueryDataset(ctx context.Context, projectID, dataset string) ([]string, error)
	SetPermissionsGroup(ctx context.Context, datasetID uuid.UUID, permissionGroupID int) error
	SoftDeleteMetadata(ctx context.Context, datasetID uuid.UUID) error
	RestoreMetadata(ctx context.Context, datasetID uuid.UUID) error
	DeleteMetadata(ctx context.Context, datasetID uuid.UUID) error
	DeleteRestrictedMetadata(ctx context.Context, datasetID uuid.UUID) error
	SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) error
}

type MetabaseAPI interface {
	AddPermissionGroupMember(ctx context.Context, groupID int, email string) error
	ArchiveCollection(ctx context.Context, colID int) error
	AutoMapSemanticTypes(ctx context.Context, dbID int) error
	CreateCollection(ctx context.Context, name string) (int, error)
	CreateCollectionWithAccess(ctx context.Context, groupIDs []int, name string) (int, error)
	CreateDatabase(ctx context.Context, team, name, saJSON, saEmail string, ds *BigQuery) (int, error)
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
	RestrictAccessToDatabase(ctx context.Context, groupIDs []int, databaseID int) error
	SetCollectionAccess(ctx context.Context, groupIDs []int, collectionID int) error
	ShowTables(ctx context.Context, ids []int) error
	Tables(ctx context.Context, dbID int) ([]MetabaseTable, error)
}

type MetabaseService interface {
	SyncTableVisibility(ctx context.Context, mbMeta *MetabaseMetadata, bq BigQuery) error
	SyncAllTablesVisibility(ctx context.Context) error
	RevokeMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error
	DeleteDatabase(ctx context.Context, dsID uuid.UUID) error
	GrantMetabaseAccess(ctx context.Context, dsID uuid.UUID, subject string) error
	MapDataset(ctx context.Context, datasetID string, services []string) (*Dataset, error)
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
	DatabaseID        int
	PermissionGroupID int
	CollectionID      int
	SAEmail           string
	DeletedAt         *time.Time
}

func CreateMetabaseMetadata(ctx context.Context, metadata MetabaseMetadata) error {
	return queries.CreateMetabaseMetadata(ctx, gensql.CreateMetabaseMetadataParams{
		DatasetID:  metadata.DatasetID,
		DatabaseID: int32(metadata.DatabaseID),
		PermissionGroupID: sql.NullInt32{
			Int32: int32(metadata.PermissionGroupID),
			Valid: metadata.PermissionGroupID > 0,
		},
		CollectionID: sql.NullInt32{
			Int32: int32(metadata.CollectionID),
			Valid: metadata.CollectionID > 0,
		},
		SaEmail: metadata.SAEmail,
	})
}

func GetMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*MetabaseMetadata, error) {
	var meta gensql.MetabaseMetadatum
	var err error
	if includeDeleted {
		meta, err = queries.GetMetabaseMetadataWithDeleted(ctx, datasetID)
	} else {
		meta, err = queries.GetMetabaseMetadata(ctx, datasetID)
	}

	if err != nil {
		return nil, err
	}

	return mbMetadataFromSQL(meta), nil
}

func GetAllMetabaseMetadata(ctx context.Context) ([]*MetabaseMetadata, error) {
	mbs, err := queries.GetAllMetabaseMetadata(ctx)
	if err != nil {
		return nil, err
	}

	mbMetas := make([]*MetabaseMetadata, len(mbs))
	for idx, meta := range mbs {
		mbMetas[idx] = mbMetadataFromSQL(meta)
	}

	return mbMetas, nil
}

func SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) error {
	return queries.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:        sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID: datasetID,
	})
}

func RestoreMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	return queries.RestoreMetabaseMetadata(ctx, datasetID)
}

func mbMetadataFromSQL(meta gensql.MetabaseMetadatum) *MetabaseMetadata {
	return &MetabaseMetadata{
		DatasetID:         meta.DatasetID,
		DatabaseID:        int(meta.DatabaseID),
		PermissionGroupID: int(meta.PermissionGroupID.Int32),
		CollectionID:      int(meta.CollectionID.Int32),
		SAEmail:           meta.SaEmail,
		DeletedAt:         nullTimeToPtr(meta.DeletedAt),
	}
}
