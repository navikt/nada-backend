package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/sqlc-dev/pqtype"
)

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

type NewBigQuery struct {
	ProjectID string  `json:"projectID"`
	Dataset   string  `json:"dataset"`
	Table     string  `json:"table"`
	PiiTags   *string `json:"piiTags"`
}

type BigquerySchema struct {
	Columns []bqclient.BigqueryColumn
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
	Metadata                 bqclient.BigqueryMetadata
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

func GetDataproducts(ctx context.Context, ids []uuid.UUID) ([]DataproductWithDataset, *APIError) {
	sqldp, err := queries.GetDataproductsWithDatasets(ctx, gensql.GetDataproductsWithDatasetsParams{
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

func GetDatasetsMinimal(ctx context.Context) ([]*DatasetMinimal, *APIError) {
	sqldss, err := queries.GetAllDatasetsMinimal(ctx)
	if err != nil {
		return nil, DBErrorToAPIError(err, "GetDatasetsMinimal(): Database error")
	}

	dss := make([]*DatasetMinimal, len(sqldss))
	for i, ds := range sqldss {
		dss[i] = &DatasetMinimal{
			ID:              ds.ID,
			Name:            ds.Name,
			Created:         ds.Created,
			BigQueryProject: ds.ProjectID,
			BigQueryDataset: ds.Dataset,
			BigQueryTable:   ds.TableName,
		}
	}

	return dss, nil
}

func GetDataset(ctx context.Context, id string) (*Dataset, *APIError) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "GetDataset(): Invalid UUID")
	}

	sqlds, err := queries.GetDatasetComplete(ctx, uuid)
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

func CreateDataproduct(ctx context.Context, input NewDataproduct) (*DataproductMinimal, *APIError) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, NewAPIError(http.StatusForbidden, err, "createDataproduct(): User not in group of dataproduct")
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	dataproduct, err := queries.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		OwnerGroup:            input.Group,
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		Slug:                  slugify(input.Slug, input.Name),
		TeamContact:           ptrToNullString(input.TeamContact),
		TeamID:                ptrToNullString(input.TeamID),
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "createDataproduct(): failed to save dataproduct")
	}

	return dataproductMinimalFromSQL(&dataproduct), nil
}

func UpdateDataproduct(ctx context.Context, id string, input UpdateDataproductDto) (*DataproductMinimal, *APIError) {
	dp, apierr := GetDataproduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, NewAPIError(http.StatusForbidden, err, "updateDataproduct(): User not in group of dataproduct")
	}
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}
	res, err := queries.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:                  input.Name,
		Description:           ptrToNullString(input.Description),
		ID:                    uuid.MustParse(id),
		OwnerTeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamContact:           ptrToNullString(input.TeamContact),
		Slug:                  slugify(input.Slug, input.Name),
		TeamID:                ptrToNullString(input.TeamID),
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "updateDataproduct(): failed to update dataproduct")
	}

	return dataproductMinimalFromSQL(&res), nil
}

func DeleteDataproduct(ctx context.Context, id string) (*DataproductWithDataset, *APIError) {
	dp, apierr := GetDataproduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, NewAPIError(http.StatusForbidden, err, "deleteDataproduct(): User not in group of dataproduct")
	}

	if err := queries.DeleteDataproduct(ctx, uuid.MustParse(id)); err != nil {
		return nil, DBErrorToAPIError(err, "deleteDataproduct(): failed to delete dataproduct")
	}

	return dp, nil
}

func dataproductMinimalFromSQL(dp *gensql.Dataproduct) *DataproductMinimal {
	return &DataproductMinimal{
		ID:           dp.ID,
		Name:         dp.Name,
		Created:      dp.Created,
		LastModified: dp.LastModified,
		Description:  &dp.Description.String,
		Slug:         dp.Slug,
		Owner: &DataproductOwner{
			Group:            dp.Group,
			TeamkatalogenURL: &dp.TeamkatalogenUrl.String,
			TeamContact:      &dp.TeamContact.String,
			TeamID:           &dp.TeamID.String,
		},
	}
}

func MapDataset(ctx context.Context, datasetID string, services []string) (*Dataset, *APIError) {
	ds, apierr := GetDataset(ctx, datasetID)
	if apierr != nil {
		return nil, apierr
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return nil, apierr
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, NewAPIError(http.StatusForbidden, err, "mapDataset(): User not in group of dataproduct")
	}

	err := queries.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: uuid.MustParse(datasetID),
		Services:  services,
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "mapDataset(): failed to map dataset")
	}

	mapMetabase := false
	for _, svc := range services {
		if svc == MappingServiceMetabase {
			mapMetabase = true
			eventManager.TriggerDatasetAddMetabaseMapping(ctx, uuid.MustParse(datasetID))
			break
		}
	}
	if !mapMetabase {
		eventManager.TriggerDatasetRemoveMetabaseMapping(ctx, uuid.MustParse(datasetID))
	}
	return ds, nil
}

func CreateDataset(ctx context.Context, input NewDataset) (*string, *APIError) {
	user := auth.GetUser(ctx)

	dp, apierr := GetDataproduct(ctx, input.DataproductID.String())
	if apierr != nil {
		return nil, apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, NewAPIError(http.StatusForbidden, err, "createDataset(): User not in group of dataproduct")
	}

	var referenceDatasource *NewBigQuery
	var pseudoBigQuery *NewBigQuery
	if len(input.PseudoColumns) > 0 {
		projectID, datasetID, tableID, err := bq.CreatePseudonymisedView(ctx, input.BigQuery.ProjectID,
			input.BigQuery.Dataset, input.BigQuery.Table, input.PseudoColumns)
		if err != nil {
			return nil, NewAPIError(http.StatusInternalServerError, err, "createDataset(): failed to create pseudonymised view")
		}

		referenceDatasource = &input.BigQuery

		pseudoBigQuery = &NewBigQuery{
			ProjectID: projectID,
			Dataset:   datasetID,
			Table:     tableID,
			PiiTags:   input.BigQuery.PiiTags,
		}
	}

	updatedInput, apierr := prepareBigQueryHandlePseudoView(ctx, input, pseudoBigQuery, dp.Owner.Group)
	if apierr != nil {
		return nil, apierr
	}

	if updatedInput.Description != nil && *updatedInput.Description != "" {
		*updatedInput.Description = html.EscapeString(*updatedInput.Description)
	}

	ds, err := dbCreateDataset(ctx, updatedInput, referenceDatasource, user)
	if err != nil {
		return nil, DBErrorToAPIError(err, "createDataset(): failed to save dataset")
	}

	if pseudoBigQuery == nil && updatedInput.GrantAllUsers != nil && *updatedInput.GrantAllUsers {
		if err := accessManager.Grant(ctx, updatedInput.BigQuery.ProjectID, updatedInput.BigQuery.Dataset, updatedInput.BigQuery.Table, "group:all-users@nav.no"); err != nil {
			return nil, NewAPIError(http.StatusInternalServerError, err, "createDataset(): failed to grant all users")
		}
	}

	return ds, nil
}

func prepareBigQueryHandlePseudoView(ctx context.Context, ds NewDataset, viewBQ *NewBigQuery, group string) (NewDataset, *APIError) {
	if err := ensureGroupOwnsGCPProject(group, ds.BigQuery.ProjectID); err != nil {
		return NewDataset{}, NewAPIError(http.StatusForbidden, err, "prepareBigQueryHandlePseudoView(): Group does not own GCP project")
	}

	if viewBQ != nil {
		metadata, err := prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, viewBQ.ProjectID, viewBQ.Dataset, viewBQ.Table)
		if err != nil {
			return NewDataset{}, err
		}
		ds.BigQuery = *viewBQ
		ds.Metadata = *metadata
		return ds, nil
	}

	metadata, err := prepareBigQuery(ctx, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.ProjectID, ds.BigQuery.Dataset, ds.BigQuery.Table)
	if err != nil {
		return NewDataset{}, err
	}
	ds.Metadata = *metadata

	return ds, nil
}

func prepareBigQuery(ctx context.Context, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable string) (*bqclient.BigqueryMetadata, *APIError) {
	metadata, err := bq.GetTableMetadata(ctx, sinkProject, sinkDataset, sinkTable)
	if err != nil {
		return nil, NewAPIError(http.StatusNotFound, err,
			fmt.Sprintf("prepareBigQuery(): failed to fetch metadata on table %v, but it does not exist in %v.%v", sinkProject, sinkDataset, sinkTable))
	}

	switch metadata.TableType {
	case bigquery.RegularTable:
	case bigquery.ViewTable:
		fallthrough
	case bigquery.MaterializedView:
		if err := accessManager.AddToAuthorizedViews(ctx, srcProject, srcDataset, sinkProject, sinkDataset, sinkTable); err != nil {
			return nil, NewAPIError(http.StatusInternalServerError, err, "prepareBigQuery(): failed to add view to authorized views")
		}
	default:
		return nil, NewAPIError(http.StatusBadRequest, nil, fmt.Sprintf("unsupported table type: %v", metadata.TableType))
	}

	return &metadata, nil
}

func dbCreateDataset(ctx context.Context, ds NewDataset, referenceDatasource *NewBigQuery, user *auth.User) (*string, error) {
	tx, err := sqldb.Begin()
	if err != nil {
		return nil, err
	}

	if ds.Keywords == nil {
		ds.Keywords = []string{}
	}

	querier := queries.WithTx(tx)
	created, err := querier.CreateDataset(ctx, gensql.CreateDatasetParams{
		Name:                     ds.Name,
		DataproductID:            ds.DataproductID,
		Description:              ptrToNullString(ds.Description),
		Pii:                      gensql.PiiLevel(ds.Pii),
		Type:                     "bigquery",
		Slug:                     slugify(ds.Slug, ds.Name),
		Repo:                     ptrToNullString(ds.Repo),
		Keywords:                 ds.Keywords,
		AnonymisationDescription: ptrToNullString(ds.AnonymisationDescription),
		TargetUser:               ptrToNullString(ds.TargetUser),
	})

	if err != nil {
		return nil, err
	}

	schemaJSON, err := json.Marshal(ds.Metadata.Schema.Columns)
	if err != nil {
		return nil, fmt.Errorf("marshalling schema: %w", err)
	}

	if ds.BigQuery.PiiTags != nil && !json.Valid([]byte(*ds.BigQuery.PiiTags)) {
		return nil, fmt.Errorf("invalid pii tags, must be json map or null: %w", err)
	}

	_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
		DatasetID:    created.ID,
		ProjectID:    ds.BigQuery.ProjectID,
		Dataset:      ds.BigQuery.Dataset,
		TableName:    ds.BigQuery.Table,
		Schema:       pqtype.NullRawMessage{RawMessage: schemaJSON, Valid: len(schemaJSON) > 4},
		LastModified: ds.Metadata.LastModified,
		Created:      ds.Metadata.Created,
		Expires:      sql.NullTime{Time: ds.Metadata.Expires, Valid: !ds.Metadata.Expires.IsZero()},
		TableType:    string(ds.Metadata.TableType),
		PiiTags: pqtype.NullRawMessage{
			RawMessage: json.RawMessage([]byte(ptrToString(ds.BigQuery.PiiTags))),
			Valid:      len(ptrToString(ds.BigQuery.PiiTags)) > 4,
		},
		PseudoColumns: ds.PseudoColumns,
		IsReference:   false,
	})

	if err != nil {
		if err := tx.Rollback(); err != nil {
			log.WithError(err).Error("Rolling back dataset and datasource_bigquery transaction")
		}
		return nil, err
	}

	if len(ds.PseudoColumns) > 0 && referenceDatasource != nil {
		_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
			DatasetID:    created.ID,
			ProjectID:    referenceDatasource.ProjectID,
			Dataset:      referenceDatasource.Dataset,
			TableName:    referenceDatasource.Table,
			Schema:       pqtype.NullRawMessage{RawMessage: schemaJSON, Valid: len(schemaJSON) > 4},
			LastModified: ds.Metadata.LastModified,
			Created:      ds.Metadata.Created,
			Expires:      sql.NullTime{Time: ds.Metadata.Expires, Valid: !ds.Metadata.Expires.IsZero()},
			TableType:    string(ds.Metadata.TableType),
			PiiTags: pqtype.NullRawMessage{
				RawMessage: json.RawMessage([]byte(ptrToString(ds.BigQuery.PiiTags))),
				Valid:      len(ptrToString(ds.BigQuery.PiiTags)) > 4,
			},
			PseudoColumns: ds.PseudoColumns,
			IsReference:   true,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.WithError(err).Error("Rolling back dataset and datasource_bigquery transaction")
			}
			return nil, err
		}
	}

	if ds.GrantAllUsers != nil && *ds.GrantAllUsers {
		_, err = querier.GrantAccessToDataset(ctx, gensql.GrantAccessToDatasetParams{
			DatasetID: created.ID,
			Expires:   sql.NullTime{},
			Subject:   emailOfSubjectToLower("group:all-users@nav.no"),
			Granter:   user.Email,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.WithError(err).Error("Rolling back dataset and datasource_bigquery transaction")
			}
			return nil, err
		}
	}

	for _, keyword := range ds.Keywords {
		err = querier.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			log.WithError(err).Warn("failed to create tag when creating dataset in database")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &created.Slug, nil
}

func DeleteDataset(ctx context.Context, id string) (string, *APIError) {
	ds, apierr := GetDataset(ctx, id)
	if apierr != nil {
		return "", apierr
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return "", apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return "", NewAPIError(http.StatusForbidden, err, "deleteDataset(): User not in group of dataproduct")
	}

	if err := queries.DeleteDataset(ctx, uuid.MustParse(id)); err != nil {
		return "", DBErrorToAPIError(err, "deleteDataset(): failed to delete dataset")
	}

	return id, nil
}

func UpdateDataset(ctx context.Context, id string, input UpdateDatasetDto) (string, *APIError) {
	ds, apierr := GetDataset(ctx, id)
	if apierr != nil {
		return "", apierr
	}

	if input.DataproductID == nil {
		input.DataproductID = &ds.DataproductID
	}

	dp, apierr := GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return "", apierr
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return "", NewAPIError(http.StatusForbidden, err, "updateDataset(): User not in group of dataproduct")
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	if *input.DataproductID != ds.DataproductID {
		dp2, err := GetDataproduct(ctx, input.DataproductID.String())
		if err != nil {
			return "", err
		}
		if err := ensureUserInGroup(ctx, dp2.Owner.Group); err != nil {
			return "", NewAPIError(http.StatusForbidden, err, "updateDataset(): User not in group of updated dataproduct")
		}
		if dp.Owner.Group != dp2.Owner.Group {
			return "", NewAPIError(http.StatusForbidden, fmt.Errorf("move dataset from dataproduct %v to %v is forbidden", dp.Name, dp2.Name),
				"cannot move dataset to a dataproduct that is not owned by the same team")
		}
	}

	if len(input.PseudoColumns) > 0 {
		referenceDatasource, err := dbGetBigqueryDatasource(ctx, id, true)
		if err != nil {
			return "", DBErrorToAPIError(err, "updateDataset(): failed to get reference datasource")
		}
		_, _, _, err = bq.CreatePseudonymisedView(ctx, referenceDatasource.ProjectID,
			referenceDatasource.Dataset, referenceDatasource.Table, input.PseudoColumns)
		if err != nil {
			return "", NewAPIError(http.StatusInternalServerError, err, "updateDataset(): failed to create pseudonymised view")
		}
	}

	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}

	updatedID, err := dbUpdateDataset(ctx, id, input)
	if err != nil {
		return "", DBErrorToAPIError(err, "updateDataset(): failed to update dataset")
	}
	return updatedID, nil
}

func dbGetBigqueryDatasource(ctx context.Context, datasetID string, isReference bool) (BigQuery, error) {
	bq, err := queries.GetBigqueryDatasource(ctx, gensql.GetBigqueryDatasourceParams{
		DatasetID:   uuid.MustParse(datasetID),
		IsReference: isReference,
	})
	if err != nil {
		return BigQuery{}, err
	}

	piiTags := "{}"
	if bq.PiiTags.RawMessage != nil {
		piiTags = string(bq.PiiTags.RawMessage)
	}

	return BigQuery{
		ID:            bq.ID,
		DatasetID:     bq.DatasetID,
		ProjectID:     bq.ProjectID,
		Dataset:       bq.Dataset,
		Table:         bq.TableName,
		TableType:     bqclient.BigQueryType(strings.ToLower(bq.TableType)),
		LastModified:  bq.LastModified,
		Created:       bq.Created,
		Expires:       nullTimeToPtr(bq.Expires),
		Description:   bq.Description.String,
		PiiTags:       &piiTags,
		MissingSince:  &bq.MissingSince.Time,
		PseudoColumns: bq.PseudoColumns,
	}, nil
}

func dbUpdateDataset(ctx context.Context, id string, input UpdateDatasetDto) (string, error) {
	if input.Keywords == nil {
		input.Keywords = []string{}
	}

	res, err := queries.UpdateDataset(ctx, gensql.UpdateDatasetParams{
		Name:                     input.Name,
		Description:              ptrToNullString(input.Description),
		ID:                       uuid.MustParse(id),
		Pii:                      gensql.PiiLevel(input.Pii),
		Slug:                     slugify(input.Slug, input.Name),
		Repo:                     ptrToNullString(input.Repo),
		Keywords:                 input.Keywords,
		DataproductID:            *input.DataproductID,
		AnonymisationDescription: ptrToNullString(input.AnonymisationDescription),
		TargetUser:               ptrToNullString(input.TargetUser),
	})
	if err != nil {
		return "", fmt.Errorf("updating dataset in database: %w", err)
	}

	//TODO: tags table should be removed
	for _, keyword := range input.Keywords {
		err = queries.CreateTagIfNotExist(ctx, keyword)
		if err != nil {
			log.WithError(err).Warn("failed to create tag when updating dataset in database")
		}
	}

	if !json.Valid([]byte(*input.PiiTags)) {
		return "", fmt.Errorf("invalid pii tags, must be json map or null: %w", err)
	}

	err = queries.UpdateBigqueryDatasource(ctx, gensql.UpdateBigqueryDatasourceParams{
		DatasetID: uuid.MustParse(id),
		PiiTags: pqtype.NullRawMessage{
			RawMessage: json.RawMessage(ptrToString(input.PiiTags)),
			Valid:      len(ptrToString(input.PiiTags)) > 4,
		},
		PseudoColumns: input.PseudoColumns,
	})
	if err != nil {
		return "", err
	}

	return res.ID.String(), nil
}

func GetAccessiblePseudoDatasetsForUser(ctx context.Context) ([]*PseudoDataset, *APIError) {
	user := auth.GetUser(ctx)
	subjectsAsOwner := []string{user.Email}
	subjectsAsOwner = append(subjectsAsOwner, user.GoogleGroups.Emails()...)
	subjectsAsAccesser := []string{"user:" + user.Email}
	for _, geml := range user.GoogleGroups.Emails() {
		subjectsAsAccesser = append(subjectsAsAccesser, "group:"+geml)
	}
	pseudoDatasets, err := dbGetAccessiblePseudoDatasourcesByUser(ctx, subjectsAsOwner, subjectsAsAccesser)
	if err != nil {
		return nil, DBErrorToAPIError(err, "getAccessiblePseudoDatasetsForUser(): failed to get accessible pseudo datasets")
	}
	return pseudoDatasets, nil
}

func dbGetAccessiblePseudoDatasourcesByUser(ctx context.Context, subjectsAsOwner []string, subjectsAsAccesser []string) ([]*PseudoDataset, error) {
	rows, err := queries.GetAccessiblePseudoDatasetsByUser(ctx, gensql.GetAccessiblePseudoDatasetsByUserParams{
		OwnerSubjects:  subjectsAsOwner,
		AccessSubjects: subjectsAsAccesser,
	})
	if err != nil {
		return nil, err
	}

	pseudoDatasets := []*PseudoDataset{}
	bqIDMap := make(map[string]int)
	for _, d := range rows {
		pseudoDataset := &PseudoDataset{
			// name is the name of the dataset
			Name: d.Name,
			// datasetID is the id of the dataset
			DatasetID: d.DatasetID,
			// datasourceID is the id of the bigquery datasource
			DatasourceID: d.BqDatasourceID,
		}
		bqID := fmt.Sprintf("%v.%v.%v", d.BqProjectID, d.BqDatasetID, d.BqTableID)

		_, exist := bqIDMap[bqID]
		if exist {
			continue
		}
		bqIDMap[bqID] = 1
		pseudoDatasets = append(pseudoDatasets, pseudoDataset)
	}
	return pseudoDatasets, nil
}

func SetDatasourceDeleted(ctx context.Context, id uuid.UUID) error {
	return queries.SetDatasourceDeleted(ctx, id)
}

func GetOwnerGroupOfDataset(ctx context.Context, datasetID uuid.UUID) (string, error) {
	return queries.GetOwnerGroupOfDataset(ctx, datasetID)
}
