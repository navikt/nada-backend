package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type PiiLevel string

type DatasourceType string

type Dataset struct {
	ID                       uuid.UUID `json:"id"`
	DataproductID            uuid.UUID `json:"dataproductID"`
	Name                     string    `json:"name"`
	Created                  time.Time `json:"created"`
	LastModified             time.Time `json:"lastModified"`
	Description              *string   `json:"description"`
	Slug                     string    `json:"slug"`
	Repo                     *string   `json:"repo"`
	Pii                      PiiLevel  `json:"pii"`
	Keywords                 []string  `json:"keywords"`
	AnonymisationDescription *string   `json:"anonymisationDescription"`
	TargetUser               *string   `json:"targetUser"`
	Access                   []*Access `json:"access"`
	Mappings                 []string  `json:"mappings"`
	Datasource               *BigQuery `json:"datasource"`
	MetabaseUrl              *string   `json:"metabaseUrl"`
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

type DataproductOwner struct {
	Group            string  `json:"group"`
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	TeamContact      *string `json:"teamContact"`
	TeamID           *string `json:"teamID"`
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

type DataproductWithDataset struct {
	Dataproduct
	Datasets []*DatasetInDataproduct `json:"datasets"`
}

func GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]DataproductWithDataset, *APIError) {
	sqldp, err := querier.GetDataproductsWithDatasets(ctx, gensql.GetDataproductsWithDatasetsParams{
		Ids:    ids,
		Groups: []string{},
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "GetDataproducts(): Database error")
	}

	return dataproductsWithDatasetFromSQL(sqldp), nil
}

func GetDataproduct(ctx context.Context, id string) (*DataproductWithDataset, *APIError) {
	dpuuid, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "GetDataproduct(): Invalid UUID")
	}
	dps, apierr := GetDataproducts(ctx, []uuid.UUID{dpuuid})
	if apierr != nil {
		return nil, apierr
	}
	// it is safe to directly use the first element without checking the length
	// because if the length was 0, the sql query in GetDataproducts should have returned no row
	return &dps[0], nil
}

func GetDatasets(ctx context.Context) ([]*Dataset, *APIError) {
	sqldss, err := querier.GetAllDatasets(ctx)
	if err != nil {
		return nil, DBErrorToAPIError(err, "GetDataset(): Database error")
	}

	var apiErr *APIError
	dss := make([]*Dataset, len(sqldss))
	for i, ds := range sqldss {
		dss[i], apiErr = datasetFromSQL([]gensql.DatasetView{ds})
		if err != nil {
			return nil, apiErr
		}
	}

	return dss, nil
}

func GetDataset(ctx context.Context, id string) (*Dataset, *APIError) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "GetDataset(): Invalid UUID")
	}

	sqlds, err := querier.GetDatasetComplete(ctx, uuid)
	if err != nil {
		return nil, DBErrorToAPIError(err, "GetDataset(): Database error")
	}

	ds, apiErr := datasetFromSQL(sqlds)
	if err != nil {
		return nil, apiErr
	}

	return ds, nil
}

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
				ID:            dsrow.DsID,
				Name:          dsrow.DsName,
				Created:       dsrow.DsCreated,
				LastModified:  dsrow.DsLastModified,
				Description:   nullStringToPtr(dsrow.DsDescription),
				Slug:          dsrow.DsSlug,
				Keywords:      dsrow.DsKeywords,
				DataproductID: dsrow.DsDpID,
				Mappings:      []string{},
				Access:        []*Access{},
				Datasource:    nil,
				Pii:           PiiLevel(dsrow.Pii),
			}
		}

		if dsrow.BqID != uuid.Nil {
			var schema []*bqclient.BigqueryColumn
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
				TableType:     bqclient.BigQueryType(dsrow.BqTableType),
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
