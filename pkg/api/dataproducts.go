package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type PiiLevel string

type DatasourceType string

type BigQueryType string

type Access struct {
	ID              uuid.UUID  `json:"id"`
	Subject         string     `json:"subject"`
	Granter         string     `json:"granter"`
	Expires         *time.Time `json:"expires"`
	Created         time.Time  `json:"created"`
	Revoked         *time.Time `json:"revoked"`
	DatasetID       uuid.UUID  `json:"datasetID"`
	AccessRequestID *uuid.UUID `json:"accessRequestID"`
}

type TableColumn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Type        string `json:"type"`
}

type BigQuery struct {
	ID            uuid.UUID
	DatasetID     uuid.UUID
	ProjectID     string         `json:"projectID"`
	Dataset       string         `json:"dataset"`
	Table         string         `json:"table"`
	TableType     BigQueryType   `json:"tableType"`
	LastModified  time.Time      `json:"lastModified"`
	Created       time.Time      `json:"created"`
	Expires       *time.Time     `json:"expired"`
	Description   string         `json:"description"`
	PiiTags       *string        `json:"piiTags"`
	MissingSince  *time.Time     `json:"missingSince"`
	PseudoColumns []string       `json:"pseudoColumns"`
	Schema        []*TableColumn `json:"schema"`
}

type DatasetDto struct {
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

type DataproductOwner struct {
	Group            string  `json:"group"`
	TeamkatalogenURL *string `json:"teamkatalogenURL"`
	TeamContact      *string `json:"teamContact"`
	TeamID           *string `json:"teamID"`
}

type DataproductDto struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Created      time.Time         `json:"created"`
	LastModified time.Time         `json:"lastModified"`
	Description  *string           `json:"description"`
	Slug         string            `json:"slug"`
	Owner        *DataproductOwner `json:"owner"`
	Datasets     []*DatasetDto     `json:"datasets"`
	Keywords     []string          `json:"keywords"`
}

func GetDataproduct(ctx context.Context, id string) (*DataproductDto, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	sqldp, err := querier.GetDataproductComplete(ctx, uuid)
	if err != nil {
		return nil, err
	}

	dp, err := dataproductsFromSQL(sqldp)
	if err != nil {
		return nil, err
	}
	if len(dp) == 0 {
		return nil, fmt.Errorf("GetDataproduct: no dataproduct with id %s", id)
	}
	return dp[0], nil
}

func dataproductsFromSQL(dprows []gensql.DataproductCompleteView) ([]*DataproductDto, error) {
	datasets, err := datasetsFromSQL(dprows)
	if err != nil {
		return nil, err
	}

	dataproducts := []*DataproductDto{}

	for _, dprow := range dprows {
		var dataproduct *DataproductDto

		for _, dp := range dataproducts {
			if dp.ID == dprow.DataproductID {
				dataproduct = dp
				break
			}
		}
		if dataproduct == nil {
			dataproduct = &DataproductDto{
				ID:           dprow.DataproductID,
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
			}
			dpdatasets := []*DatasetDto{}
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
	}
	return dataproducts, nil
}

func datasetsFromSQL(dsrows []gensql.DataproductCompleteView) ([]*DatasetDto, error) {
	datasets := []*DatasetDto{}

	for _, dsrow := range dsrows {
		if !dsrow.DsID.Valid {
			continue
		}

		piiTags := "{}"
		if dsrow.PiiTags.RawMessage != nil {
			piiTags = string(dsrow.PiiTags.RawMessage)
		}

		var ds *DatasetDto

		for _, dsIn := range datasets {
			if dsIn.ID == dsrow.DsID.UUID {
				ds = dsIn
				break
			}
		}
		if ds == nil {
			ds = &DatasetDto{
				ID:            dsrow.DsID.UUID,
				Name:          dsrow.DsName.String,
				Created:       dsrow.DsCreated.Time,
				LastModified:  dsrow.DsLastModified.Time,
				Description:   nullStringToPtr(dsrow.DsDescription),
				Slug:          dsrow.DsSlug.String,
				Keywords:      dsrow.DsKeywords,
				DataproductID: dsrow.DataproductID,
				Mappings:      []string{},
				Access:        []*Access{},
				Datasource:    nil,
			}
			datasets = append(datasets, ds)
		}

		if dsrow.BqID != uuid.Nil {
			var schema []*TableColumn
			if dsrow.BqSchema.Valid {
				if err := json.Unmarshal(dsrow.BqSchema.RawMessage, &schema); err != nil {
					return nil, fmt.Errorf("unmarshalling schema: %w", err)
				}
			}

			dsrc := &BigQuery{
				ID:            dsrow.BqID,
				DatasetID:     dsrow.DsID.UUID,
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
			ds.Datasource = dsrc
		}

		if len(dsrow.MappingServices) > 0 {
			for _, service := range dsrow.MappingServices {
				exist := false
				for _, mapping := range ds.Mappings {
					if mapping == service {
						exist = true
						break
					}
				}
				if !exist {
					ds.Mappings = append(ds.Mappings, service)
				}
			}
		}

		if dsrow.AccessID.Valid {
			exist := false
			for _, dsAccess := range ds.Access {
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
					DatasetID:       dsrow.DsID.UUID,
					AccessRequestID: nullUUIDToUUIDPtr(dsrow.AccessRequestID),
				}
				ds.Access = append(ds.Access, access)
			}
		}

		if ds.MetabaseUrl == nil && dsrow.MbDatabaseID.Valid {
			base := "https://metabase.intern.dev.nav.no/browse/%v"
			if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
				base = "https://metabase.intern.nav.no/browse/%v"
			}
			url := fmt.Sprintf(base, dsrow.MbDatabaseID.Int32)
			ds.MetabaseUrl = &url
		}
	}

	return datasets, nil
}
