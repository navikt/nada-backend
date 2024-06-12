package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type DataProductsStorage interface {
	GetDataproduct(ctx context.Context, id string) (*DataproductWithDataset, error)
	GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]DataproductWithDataset, error)
	GetDataset(ctx context.Context, id string) (*Dataset, error)
	DeleteDataset(ctx context.Context, id uuid.UUID) error
	SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error
	GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error)
	CreateDataproduct(ctx context.Context, input NewDataproduct) (*DataproductMinimal, error)
	UpdateDataproduct(ctx context.Context, id string, input UpdateDataproductDto) (*DataproductMinimal, error)
	DeleteDataproduct(ctx context.Context, id string) error
	CreateDataset(ctx context.Context, ds NewDataset, referenceDatasource *NewBigQuery, user *auth.User) (*string, error)
	UpdateDataset(ctx context.Context, id string, input UpdateDatasetDto) (string, error)
	GetAccessiblePseudoDatasourcesByUser(ctx context.Context, subjectsAsOwner []string, subjectsAsAccesser []string) ([]*PseudoDataset, error)
	GetDatasetsMinimal(ctx context.Context) ([]*DatasetMinimal, error)
}

type DataProductsService interface {
	CreateDataproduct(ctx context.Context, input NewDataproduct) (*DataproductMinimal, error)
	UpdateDataproduct(ctx context.Context, id string, input UpdateDataproductDto) (*DataproductMinimal, error)
	DeleteDataproduct(ctx context.Context, id string) (*DataproductWithDataset, error)
	CreateDataset(ctx context.Context, input NewDataset) (*string, error)
	DeleteDataset(ctx context.Context, id string) (string, error)
	UpdateDataset(ctx context.Context, id string, input UpdateDatasetDto) (string, error)
	GetAccessiblePseudoDatasetsForUser(ctx context.Context) ([]*PseudoDataset, error)
	GetDatasetsMinimal(ctx context.Context) ([]*DatasetMinimal, error)
}

type PiiLevel string

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
	TeamID           *string    `json:"teamID"`
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
	ProductAreaID *string `json:"productAreaID,omitempty"`
	// Id of the team.
	TeamID *string `json:"teamID,omitempty"`
	Slug   *string
}

type UpdateDataproductDto struct {
	Name             string   `json:"name"`
	Description      *string  `json:"description"`
	Slug             *string  `json:"slug"`
	Pii              PiiLevel `json:"pii"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	TeamContact      *string  `json:"teamContact"`
	ProductAreaID    *string  `json:"productAreaID"`
	TeamID           *string  `json:"teamID"`
}

const (
	MappingServiceMetabase string = "metabase"
)

func dataproductsWithDatasetFromSQL(dprows []gensql.GetDataproductsWithDatasetsRow) []DataproductWithDataset {
	if dprows == nil {
		return []DataproductWithDataset{}
	}

	datasets := datasetsInDataProductFromSQL(dprows)

	dataproducts := []DataproductWithDataset{}

__loop_rows:
	for _, dprow := range dprows {
		for _, dp := range dataproducts {
			if dp.ID == dprow.DpID {
				continue __loop_rows
			}
		}
		dataproduct := DataproductWithDataset{
			Dataproduct: Dataproduct{
				ID:           dprow.DpID,
				Name:         dprow.DpName,
				Created:      dprow.DpCreated,
				LastModified: dprow.DpLastModified,
				Description:  nullStringToPtr(dprow.DpDescription),
				Slug:         dprow.DpSlug,
				Owner: &DataproductOwner{
					Group:            dprow.DpGroup,
					TeamkatalogenURL: nullStringToPtr(dprow.TeamkatalogenUrl),
					TeamContact:      nullStringToPtr(dprow.TeamContact),
					TeamID:           nullStringToPtr(dprow.TeamID),
					ProductAreaID:    nullUUIDToUUIDPtr(dprow.PaID),
				},
			},
		}
		dpdatasets := []*DatasetInDataproduct{}
		for _, ds := range datasets {
			if ds.DataproductID == dataproduct.ID {
				dpdatasets = append(dpdatasets, ds)
			}
		}

		keywordsMap := make(map[string]bool)
		for _, ds := range dpdatasets {
			for _, k := range ds.Keywords {
				keywordsMap[k] = true
			}
		}
		keywords := []string{}
		for k := range keywordsMap {
			keywords = append(keywords, k)
		}

		dataproduct.Datasets = dpdatasets
		dataproduct.Keywords = keywords
		dataproducts = append(dataproducts, dataproduct)
	}
	return dataproducts
}

func dataproductsWithDatasetAndAccessRequestsForGranterFromSQL(dprrows []gensql.GetDataproductsWithDatasetsAndAccessRequestsRow) ([]DataproductWithDataset, []AccessRequestForGranter, *APIError) {
	if dprrows == nil {
		return nil, nil, nil
	}

	dprows := make([]gensql.GetDataproductsWithDatasetsRow, len(dprrows))
	for i, dprrow := range dprrows {
		dprows[i] = gensql.GetDataproductsWithDatasetsRow{
			DpID:             dprrow.DpID,
			DpName:           dprrow.DpName,
			DpCreated:        dprrow.DpCreated,
			DpLastModified:   dprrow.DpLastModified,
			DpDescription:    dprrow.DpDescription,
			DpSlug:           dprrow.DpSlug,
			DpGroup:          dprrow.DpGroup,
			TeamkatalogenUrl: dprrow.TeamkatalogenUrl,
			TeamContact:      dprrow.TeamContact,
			TeamID:           dprrow.TeamID,
		}
	}
	dp := dataproductsWithDatasetFromSQL(dprows)

	arrows := make([]gensql.DatasetAccessRequest, 0)

	for _, dprrow := range dprrows {
		if dprrow.DarID.Valid {
			arrows = append(arrows, gensql.DatasetAccessRequest{
				ID:                   dprrow.DarID.UUID,
				DatasetID:            dprrow.DarDatasetID.UUID,
				Subject:              dprrow.DarSubject.String,
				Created:              dprrow.DarCreated.Time,
				Status:               dprrow.DarStatus.AccessRequestStatusType,
				Closed:               dprrow.DarClosed,
				Expires:              dprrow.DarExpires,
				Granter:              dprrow.DarGranter,
				Owner:                dprrow.DarOwner.String,
				PollyDocumentationID: dprrow.DarPollyDocumentationID,
				Reason:               dprrow.DarReason,
			})
		}
	}
	ars, err := AccessRequestsFromSQL(context.Background(), arrows)

	arfg := make([]AccessRequestForGranter, len(ars))
	for i, ar := range ars {
		dataproductID := uuid.Nil
		datasetName := ""
		dataproductName := ""
		dataproductSlug := ""
		for _, dprrow := range dprrows {
			if dprrow.DarDatasetID.UUID == ar.DatasetID {
				dataproductID = dprrow.DpID
				datasetName = dprrow.DsName.String
				dataproductName = dprrow.DpName
				dataproductSlug = dprrow.DpSlug
				break
			}
		}

		arfg[i] = AccessRequestForGranter{
			AccessRequest:   ar,
			DatasetName:     datasetName,
			DataproductName: dataproductName,
			DataproductID:   dataproductID,
			DataproductSlug: dataproductSlug,
		}
	}
	if err != nil {
		return nil, nil, NewAPIError(http.StatusInternalServerError, err, "dataproductsWithDatasetAndAccessRequestsFromSQL(): Error in accessRequestsFromSQL")
	}

	return dp, arfg, nil
}

func datasetsInDataProductFromSQL(dsrows []gensql.GetDataproductsWithDatasetsRow) []*DatasetInDataproduct {
	datasets := []*DatasetInDataproduct{}

	for _, dsrow := range dsrows {
		if !dsrow.DsID.Valid {
			continue
		}

		var ds *DatasetInDataproduct

		for _, dsIn := range datasets {
			if dsIn.ID == dsrow.DsID.UUID {
				ds = dsIn
				break
			}
		}
		if ds == nil {
			ds = &DatasetInDataproduct{
				ID:                     dsrow.DsID.UUID,
				Name:                   dsrow.DsName.String,
				Created:                dsrow.DsCreated.Time,
				LastModified:           dsrow.DsLastModified.Time,
				Description:            nullStringToPtr(dsrow.DsDescription),
				Slug:                   dsrow.DsSlug.String,
				Keywords:               dsrow.DsKeywords,
				DataproductID:          dsrow.DpID,
				DataSourceLastModified: dsrow.DsrcLastModified.Time,
			}
			datasets = append(datasets, ds)
		}
	}

	return datasets
}

func datasetFromSQL(dsrows []gensql.DatasetView) (*Dataset, *APIError) {
	var dataset *Dataset

	for _, dsrow := range dsrows {
		piiTags := "{}"
		if dsrow.PiiTags.RawMessage != nil {
			piiTags = string(dsrow.PiiTags.RawMessage)
		}
		if dataset == nil {
			dataset = &Dataset{
				ID:                dsrow.DsID,
				Name:              dsrow.DsName,
				Created:           dsrow.DsCreated,
				LastModified:      dsrow.DsLastModified,
				Description:       nullStringToPtr(dsrow.DsDescription),
				Slug:              dsrow.DsSlug,
				Keywords:          dsrow.DsKeywords,
				DataproductID:     dsrow.DsDpID,
				Mappings:          []string{},
				Access:            []*Access{},
				Datasource:        nil,
				Pii:               PiiLevel(dsrow.Pii),
				MetabaseDeletedAt: nullTimeToPtr(dsrow.MbDeletedAt),
			}
		}

		if dsrow.BqID != uuid.Nil {
			var schema []*BigqueryColumn
			if dsrow.BqSchema.Valid {
				if err := json.Unmarshal(dsrow.BqSchema.RawMessage, &schema); err != nil {
					return nil, NewAPIError(http.StatusInternalServerError, err, "datasetFromSQL(): Error in BigQuery schema")
				}
			}

			dsrc := &BigQuery{
				ID:            dsrow.BqID,
				DatasetID:     dsrow.DsID,
				ProjectID:     dsrow.BqProject,
				Dataset:       dsrow.BqDataset,
				Table:         dsrow.BqTableName,
				TableType:     BigQueryType(dsrow.BqTableType),
				Created:       dsrow.BqCreated,
				LastModified:  dsrow.BqLastModified,
				Expires:       nullTimeToPtr(dsrow.BqExpires),
				Description:   dsrow.BqDescription.String,
				PiiTags:       &piiTags,
				MissingSince:  nullTimeToPtr(dsrow.BqMissingSince),
				PseudoColumns: dsrow.PseudoColumns,
				Schema:        schema,
			}
			dataset.Datasource = dsrc
		}

		if len(dsrow.MappingServices) > 0 {
			for _, service := range dsrow.MappingServices {
				exist := false
				for _, mapping := range dataset.Mappings {
					if mapping == service {
						exist = true
						break
					}
				}
				if !exist {
					dataset.Mappings = append(dataset.Mappings, service)
				}
			}
		}

		if dsrow.AccessID.Valid {
			exist := false
			for _, dsAccess := range dataset.Access {
				if dsAccess.ID == dsrow.AccessID.UUID {
					exist = true
					break
				}
			}
			if !exist {
				access := &Access{
					ID:              dsrow.AccessID.UUID,
					Subject:         dsrow.AccessSubject.String,
					Granter:         dsrow.AccessGranter.String,
					Expires:         nullTimeToPtr(dsrow.AccessExpires),
					Created:         dsrow.AccessCreated.Time,
					Revoked:         nullTimeToPtr(dsrow.AccessRevoked),
					DatasetID:       dsrow.DsID,
					AccessRequestID: nullUUIDToUUIDPtr(dsrow.AccessRequestID),
				}
				dataset.Access = append(dataset.Access, access)
			}
		}

		if dataset.MetabaseUrl == nil && dsrow.MbDatabaseID.Valid {
			base := "https://metabase.intern.dev.nav.no/browse/databases/%v"
			if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
				base = "https://metabase.intern.nav.no/browse/databases/%v"
			}
			url := fmt.Sprintf(base, dsrow.MbDatabaseID.Int32)
			dataset.MetabaseUrl = &url
		}
	}

	return dataset, nil
}
