package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DataProductsStorage interface {
	CreateDataproduct(ctx context.Context, input NewDataproduct) (*DataproductMinimal, error)
	CreateDataset(ctx context.Context, ds NewDataset, referenceDatasource *NewBigQuery, user *User) (*Dataset, error)
	DeleteDataproduct(ctx context.Context, id uuid.UUID) error
	DeleteDataset(ctx context.Context, id uuid.UUID) error
	GetAccessibleDatasets(ctx context.Context, userGroups []string, requester string) (owned []*AccessibleDataset, granted []*AccessibleDataset, serviceAccountGranted []*AccessibleDataset, err error)
	GetAccessiblePseudoDatasourcesByUser(ctx context.Context, subjectsAsOwner []string, subjectsAsAccesser []string) ([]*PseudoDataset, error)
	GetDataproduct(ctx context.Context, id uuid.UUID) (*DataproductWithDataset, error)
	GetDataproductKeywords(ctx context.Context, dpid uuid.UUID) ([]string, error)
	GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]DataproductWithDataset, error)
	GetDataproductsByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*Dataproduct, error)
	GetDataproductsNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
	GetDataproductsWithDatasetsAndAccessRequests(ctx context.Context, ids []uuid.UUID, groups []string) ([]DataproductWithDataset, []AccessRequestForGranter, error)
	GetDataset(ctx context.Context, id uuid.UUID) (*Dataset, error)
	GetDatasetsMinimal(ctx context.Context) ([]*DatasetMinimal, error)
	GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error)
	SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error
	UpdateDataproduct(ctx context.Context, id uuid.UUID, input UpdateDataproductDto) (*DataproductMinimal, error)
	UpdateDataset(ctx context.Context, id uuid.UUID, input UpdateDatasetDto) (string, error)
}

type DataProductsService interface {
	CreateDataproduct(ctx context.Context, user *User, input NewDataproduct) (*DataproductMinimal, error)
	UpdateDataproduct(ctx context.Context, user *User, id uuid.UUID, input UpdateDataproductDto) (*DataproductMinimal, error)
	DeleteDataproduct(ctx context.Context, user *User, id uuid.UUID) (*DataproductWithDataset, error)
	CreateDataset(ctx context.Context, user *User, input NewDataset) (*Dataset, error)
	DeleteDataset(ctx context.Context, user *User, id uuid.UUID) (string, error)
	UpdateDataset(ctx context.Context, user *User, id uuid.UUID, input UpdateDatasetDto) (string, error)
	GetDataset(ctx context.Context, id uuid.UUID) (*Dataset, error)
	GetAccessiblePseudoDatasetsForUser(ctx context.Context, user *User) ([]*PseudoDataset, error)
	GetDatasetsMinimal(ctx context.Context) ([]*DatasetMinimal, error)
	GetDataproduct(ctx context.Context, id uuid.UUID) (*DataproductWithDataset, error)
}

type PiiLevel string

const (
	PiiLevelSensitive  PiiLevel = "sensitive"
	PiiLevelAnonymised PiiLevel = "anonymised"
	PiiLevelNone       PiiLevel = "none"
)

type DatasourceType string

type Dataset struct {
	ID                       uuid.UUID  `json:"id"`
	DataproductID            uuid.UUID  `json:"dataproductID"`
	Name                     string     `json:"name"`
	Created                  time.Time  `json:"created"`
	LastModified             time.Time  `json:"lastModified"`
	Description              *string    `json:"description"`
	Slug                     string     `json:"slug"`
	Repo                     *string    `json:"repo"`
	Pii                      PiiLevel   `json:"pii"`
	Keywords                 []string   `json:"keywords"`
	AnonymisationDescription *string    `json:"anonymisationDescription"`
	TargetUser               *string    `json:"targetUser"`
	Access                   []*Access  `json:"access"`
	Mappings                 []string   `json:"mappings"`
	Datasource               *BigQuery  `json:"datasource"`
	MetabaseUrl              *string    `json:"metabaseUrl"`
	MetabaseDeletedAt        *time.Time `json:"metabaseDeletedAt"`
}

type AccessibleDataset struct {
	Dataset
	DataproductName string  `json:"dataproductName"`
	Slug            string  `json:"slug"`
	DpSlug          string  `json:"dpSlug"`
	Group           string  `json:"group"`
	Subject         *string `json:"subject"`
}

type AccessibleDatasets struct {
	// owned
	Owned []*AccessibleDataset `json:"owned"`
	// granted
	Granted []*AccessibleDataset `json:"granted"`
	// service account granted
	ServiceAccountGranted []*AccessibleDataset `json:"serviceAccountGranted"`
}

type DatasetMinimal struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Created         time.Time `json:"created"`
	BigQueryProject string    `json:"project"`
	BigQueryDataset string    `json:"dataset"`
	BigQueryTable   string    `json:"table"`
}

type DatasetInDataproduct struct {
	ID                     uuid.UUID `json:"id"`
	DataproductID          uuid.UUID `json:"-"`
	Name                   string    `json:"name"`
	Created                time.Time `json:"created"`
	LastModified           time.Time `json:"lastModified"`
	Description            *string   `json:"description"`
	Slug                   string    `json:"slug"`
	Keywords               []string  `json:"keywords"`
	DataSourceLastModified time.Time `json:"dataSourceLastModified"`
}

type NewDataset struct {
	DataproductID            uuid.UUID   `json:"dataproductID"`
	Name                     string      `json:"name"`
	Description              *string     `json:"description"`
	Slug                     *string     `json:"slug"`
	Repo                     *string     `json:"repo"`
	Pii                      PiiLevel    `json:"pii"`
	Keywords                 []string    `json:"keywords"`
	BigQuery                 NewBigQuery `json:"bigquery"`
	AnonymisationDescription *string     `json:"anonymisationDescription"`
	GrantAllUsers            *bool       `json:"grantAllUsers"`
	TargetUser               *string     `json:"targetUser"`
	Metadata                 BigqueryMetadata
	PseudoColumns            []string `json:"pseudoColumns"`
}

type UpdateDatasetDto struct {
	Name                     string     `json:"name"`
	Description              *string    `json:"description"`
	Slug                     *string    `json:"slug"`
	Repo                     *string    `json:"repo"`
	Pii                      PiiLevel   `json:"pii"`
	Keywords                 []string   `json:"keywords"`
	DataproductID            *uuid.UUID `json:"dataproductID"`
	AnonymisationDescription *string    `json:"anonymisationDescription"`
	PiiTags                  *string    `json:"piiTags"`
	TargetUser               *string    `json:"targetUser"`
	PseudoColumns            []string   `json:"pseudoColumns"`
}

type DataproductOwner struct {
	Group            string     `json:"group"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	TeamContact      *string    `json:"teamContact"`
	TeamID           *uuid.UUID `json:"teamID"`
	ProductAreaID    *uuid.UUID `json:"productAreaID"`
}

type Dataproduct struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Created         time.Time         `json:"created"`
	LastModified    time.Time         `json:"lastModified"`
	Description     *string           `json:"description"`
	Slug            string            `json:"slug"`
	Owner           *DataproductOwner `json:"owner"`
	Keywords        []string          `json:"keywords"`
	TeamName        *string           `json:"teamName"`
	ProductAreaName string            `json:"productAreaName"`
}

type DataproductMinimal struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Created      time.Time         `json:"created"`
	LastModified time.Time         `json:"lastModified"`
	Description  *string           `json:"description"`
	Slug         string            `json:"slug"`
	Owner        *DataproductOwner `json:"owner"`
}

type DataproductWithDataset struct {
	Dataproduct
	Datasets []*DatasetInDataproduct `json:"datasets"`
}

type DatasetMap struct {
	Services []string `json:"services"`
}

// PseudoDataset contains information about a pseudo dataset
type PseudoDataset struct {
	// name is the name of the dataset
	Name string `json:"name"`
	// datasetID is the id of the dataset
	DatasetID uuid.UUID `json:"datasetID"`
	// datasourceID is the id of the bigquery datasource
	DatasourceID uuid.UUID `json:"datasourceID"`
}

// NewDataproduct contains metadata for creating a new dataproduct
type NewDataproduct struct {
	// name of dataproduct
	Name string `json:"name"`
	// description of the dataproduct
	Description *string `json:"description,omitempty"`
	// owner group email for the dataproduct.
	Group string `json:"group"`
	// owner Teamkatalogen URL for the dataproduct.
	TeamkatalogenURL *string `json:"teamkatalogenURL,omitempty"`
	// The contact information of the team who owns the dataproduct, which can be slack channel, slack account, email, and so on.
	TeamContact *string `json:"teamContact,omitempty"`
	// Id of the team's product area.
	ProductAreaID *uuid.UUID `json:"productAreaID,omitempty"`
	// Id of the team.
	TeamID *uuid.UUID `json:"teamID,omitempty"`
	Slug   *string
}

type UpdateDataproductDto struct {
	Name             string     `json:"name"`
	Description      *string    `json:"description"`
	Slug             *string    `json:"slug"`
	Pii              PiiLevel   `json:"pii"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	TeamContact      *string    `json:"teamContact"`
	ProductAreaID    *uuid.UUID `json:"productAreaID"`
	TeamID           *uuid.UUID `json:"teamID"`
}

const (
	MappingServiceMetabase string = "metabase"
)
