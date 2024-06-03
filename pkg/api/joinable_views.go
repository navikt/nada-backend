package api

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bqclient"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

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

func getJoinableViewsForReferenceAndUser(ctx context.Context, user string, pseudoDatasetID uuid.UUID) ([]gensql.GetJoinableViewsForReferenceAndUserRow, error) {
	joinableViews, err := queries.GetJoinableViewsForReferenceAndUser(ctx, gensql.GetJoinableViewsForReferenceAndUserParams{
		Owner:           user,
		PseudoDatasetID: pseudoDatasetID,
	})
	if err != nil {
		return nil, err
	}

	return joinableViews, nil
}

func getJoinableViewsForUser(ctx context.Context) ([]JoinableView, *APIError) {
	user := auth.GetUser(ctx)
	joinableViewsDB, err := queries.GetJoinableViewsForOwner(ctx, user.Email)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to get joinable views for user")
	}

	joinableViewsDBMerged := make(map[uuid.UUID][]gensql.GetJoinableViewsForOwnerRow)
	for _, vdb := range joinableViewsDB {
		_, _exist := joinableViewsDBMerged[vdb.ID]
		if !_exist {
			joinableViewsDBMerged[vdb.ID] = []gensql.GetJoinableViewsForOwnerRow{}
		}
		joinableViewsDBMerged[vdb.ID] = append(joinableViewsDBMerged[vdb.ID], vdb)
	}

	joinableViews := []JoinableView{}

	for k, v := range joinableViewsDBMerged {
		newJoinableView := JoinableView{
			ID:      k,
			Name:    v[0].Name,
			Created: v[0].Created,
			Expires: nullTimeToPtr(v[0].Expires),
		}
		joinableViews = append(joinableViews, newJoinableView)
	}

	return joinableViews, nil
}

func getJoinableView(ctx context.Context, id string) (*JoinableViewWithDatasource, *APIError) {
	user := auth.GetUser(ctx)
	joinableViewID, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "Invalid joinable view ID")
	}
	joinableViewDatasets, err := queries.GetJoinableViewWithDataset(ctx, joinableViewID)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to get joinable view")
	}

	jv := JoinableViewWithDatasource{
		JoinableView: JoinableView{
			ID:      joinableViewDatasets[0].JoinableViewID,
			Name:    joinableViewDatasets[0].JoinableViewName,
			Created: joinableViewDatasets[0].JoinableViewCreated,
			Expires: nullTimeToPtr(joinableViewDatasets[0].JoinableViewExpires),
		},
	}

	for _, ds := range joinableViewDatasets {
		jvbq := &PseudoDatasource{
			BigQueryUrl: bq.MakeBigQueryUrlForJoinableViews(jv.Name, ds.BqTable, ds.BqProject, ds.BqDataset),
			Deleted:     ds.Deleted.Valid,
		}
		jv.PseudoDatasources = append(jv.PseudoDatasources, *jvbq)

		if jvbq.Deleted {
			continue
		}

		if user.GoogleGroups.Contains(ds.Group.String) {
			jvbq.Accessible = true
		} else {
			activeAccessList, apierr := listActiveAccessToDataset(ctx, ds.DatasetID.UUID)
			if apierr != nil {
				return nil, apierr
			}

			jvbq.Accessible = false
			for _, access := range activeAccessList {
				if access.Subject == "user:"+user.Email {
					jvbq.Accessible = true
					break
				}
			}
		}

	}

	return &jv, nil

}

func createJoinableViews(ctx context.Context, input NewJoinableViews) (string, *APIError) {
	user := auth.GetUser(ctx)
	datasets := []*Dataset{}
	for _, dsid := range input.DatasetIDs {
		dataset, apierr := getDataset(ctx, dsid.String())
		if apierr != nil {
			return "", apierr
		}
		dataproduct, apierr := getDataproduct(ctx, dataset.DataproductID.String())
		if apierr != nil {
			return "", apierr
		}
		if !user.GoogleGroups.Contains(dataproduct.Owner.Group) {
			access, apierr := listActiveAccessToDataset(ctx, dataset.ID)
			if apierr != nil {
				return "", apierr
			}
			accessSet := make(map[string]int)
			for _, da := range access {
				accessSet[da.Subject] = 1
			}
			for _, ugg := range user.GoogleGroups {
				accessSet["group:"+ugg.Email] = 1
			}
			accessSet["user:"+user.Email] = 1
			if len(accessSet) == len(user.GoogleGroups.Emails())+1+len(access) {
				return "", NewAPIError(http.StatusForbidden, nil, "Access denied")
			}
		}
		datasets = append(datasets, dataset)
	}

	datasources := []bqclient.JoinableViewDatasource{}
	pseudoDatasourceIDs := []uuid.UUID{}
	for _, ds := range datasets {
		var refDatasource *BigQuery
		var pseudoDatasource *BigQuery
		var apierr *APIError
		if refDatasource, apierr = getBigqueryDatasource(ctx, ds.ID, true); apierr != nil {
			return "", apierr
		}

		if pseudoDatasource, apierr = getBigqueryDatasource(ctx, ds.ID, false); apierr != nil {
			return "", NewAPIError(http.StatusInternalServerError, apierr, "failed to find bigquery datasource")
		}
		datasources = append(datasources, bqclient.JoinableViewDatasource{
			RefDatasource: &bqclient.DatasourceForJoinableView{
				Project: refDatasource.ProjectID,
				Dataset: refDatasource.Dataset,
				Table:   refDatasource.Table,
			},
			PseudoDatasource: &bqclient.DatasourceForJoinableView{
				Project: pseudoDatasource.ProjectID,
				Dataset: pseudoDatasource.Dataset,
				Table:   pseudoDatasource.Table,
			},
		})
		pseudoDatasourceIDs = append(pseudoDatasourceIDs, pseudoDatasource.ID)
	}

	projectID, joinableDatasetID, joinableViewsMap, err := bq.CreateJoinableViewsForUser(ctx, input.Name, datasources)
	if err != nil {
		return "", NewAPIError(http.StatusInternalServerError, err, "Failed to create joinable views")
	}

	for _, d := range datasources {
		dstbl := d.RefDatasource
		if err := accessManager.AddToAuthorizedViews(ctx, dstbl.Project, dstbl.Dataset, projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", NewAPIError(http.StatusInternalServerError, err, "Failed to add to authorized views")
		}
		if err := accessManager.AddToAuthorizedViews(ctx, projectID, "secrets_vault", projectID, joinableDatasetID, joinableViewsMap[dstbl.Dataset]); err != nil {
			return "", NewAPIError(http.StatusInternalServerError, err, "Failed to add to secrets' authorized views")
		}
	}

	subj := user.Email
	subjType := models.SubjectTypeUser
	subjWithType := subjType.String() + ":" + subj

	for _, v := range joinableViewsMap {
		if err := accessManager.Grant(ctx, projectID, joinableDatasetID, v, subjWithType); err != nil {
			return "", NewAPIError(http.StatusInternalServerError, err, "Failed to grant access")
		}
	}

	if _, err := createJoinableViewsDB(ctx, joinableDatasetID, user.Email, input.Expires, pseudoDatasourceIDs); err != nil {
		return "", DBErrorToAPIError(err, "Failed to create joinable views")
	}

	return joinableDatasetID, nil
}

func createJoinableViewsDB(ctx context.Context, name, owner string, expires *time.Time, datasourceIDs []uuid.UUID) (string, error) {
	tx, err := sqldb.Begin()
	if err != nil {
		return "", err
	}

	jv, err := queries.CreateJoinableViews(ctx, gensql.CreateJoinableViewsParams{
		Name:    name,
		Owner:   owner,
		Created: time.Now(),
		Expires: ptrToNullTime(expires),
	})
	if err != nil {
		return "", err
	}
	for _, bqid := range datasourceIDs {
		if err != nil {
			return "", err
		}

		_, err = queries.CreateJoinableViewsDatasource(ctx, gensql.CreateJoinableViewsDatasourceParams{
			JoinableViewID: jv.ID,
			DatasourceID:   bqid,
		})

		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return jv.ID.String(), nil
}
