package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type JoinableViewsStorage interface {
	GetJoinableViewsForReferenceAndUser(ctx context.Context, user string, pseudoDatasetID uuid.UUID) ([]JoinableViewForReferenceAndUser, error)
	GetJoinableViewsForOwner(ctx context.Context, user *User) ([]JoinableViewForOwner, error)
	GetJoinableViewWithDataset(ctx context.Context, id uuid.UUID) ([]JoinableViewWithDataset, error)
	CreateJoinableViewsDB(ctx context.Context, name, owner string, expires *time.Time, datasourceIDs []uuid.UUID) (string, error)
	GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]JoinableViewToBeDeletedWithRefDatasource, error)
	GetJoinableViewsWithReference(ctx context.Context) ([]JoinableViewWithReference, error)
	SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error
}

type JoinableViewsService interface {
	GetJoinableViewsForUser(ctx context.Context, user *User) ([]JoinableView, error)
	GetJoinableView(ctx context.Context, user *User, id uuid.UUID) (*JoinableViewWithDatasource, error)
	CreateJoinableViews(ctx context.Context, user *User, input NewJoinableViews) (string, error)
	GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]JoinableViewToBeDeletedWithRefDatasource, error)
	GetJoinableViewsWithReference(ctx context.Context) ([]JoinableViewWithReference, error)
	SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error
}

type JoinableViewToBeDeletedWithRefDatasource struct {
	JoinableViewID   uuid.UUID
	JoinableViewName string
	BqProjectID      string
	BqDatasetID      string
	BqTableID        string
}

type JoinableViewWithReference struct {
	Owner               string
	JoinableViewID      uuid.UUID
	JoinableViewDataset string
	PseudoViewID        uuid.UUID
	PseudoProjectID     string
	PseudoDataset       string
	PseudoTable         string
	Expires             sql.NullTime
}

type JoinableViewWithDataset struct {
	BqProject           string
	BqDataset           string
	BqTable             string
	Deleted             *time.Time
	DatasetID           uuid.NullUUID
	JoinableViewID      uuid.UUID
	Group               string
	JoinableViewName    string
	JoinableViewCreated time.Time
	JoinableViewExpires *time.Time
}

type JoinableViewForReferenceAndUser struct {
	ID      uuid.UUID
	Dataset string
}

type JoinableViewForOwner struct {
	ID        uuid.UUID
	Name      string
	Owner     string
	Created   time.Time
	Expires   *time.Time
	ProjectID string
	DatasetID string
	TableID   string
}

// NewJoinableViews contains metadata for creating joinable views
type NewJoinableViews struct {
	// Name is the name of the joinable views which will be used as the name of the dataset in bigquery, which contains all the joinable views
	Name    string     `json:"name"`
	Expires *time.Time `json:"expires"`
	// DatasetIDs is the IDs of the datasets which are made joinable.
	DatasetIDs []uuid.UUID `json:"datasetIDs"`
}

type JoinableView struct {
	// id is the id of the joinable view set
	ID      uuid.UUID  `json:"id"`
	Name    string     `json:"name"`
	Created time.Time  `json:"created"`
	Expires *time.Time `json:"expires"`
}

type PseudoDatasource struct {
	BigQueryUrl string `json:"bigqueryUrl"`
	Accessible  bool   `json:"accessible"`
	Deleted     bool   `json:"deleted"`
}

type JoinableViewWithDatasource struct {
	JoinableView
	PseudoDatasources []PseudoDatasource `json:"pseudoDatasources"`
}
