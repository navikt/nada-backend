package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type JoinableView struct {
	ID                uuid.UUID
	Name              string
	Created           time.Time
	Expires           *time.Time
	PseudoDatasources []*models.BigQuery
}

type JoinableViewInDetail struct {
	JoinableView
	AccessToViews []bool
}

func (r *Repo) GetJoinableViewsForUser(ctx context.Context, user string) ([]*JoinableView, error) {
	joinableViewsDB, err := r.querier.GetJoinableViewsForOwner(ctx, user)
	if err != nil {
		return nil, err
	}

	joinableViewsDBMerged := make(map[uuid.UUID][]gensql.GetJoinableViewsForOwnerRow)
	for _, vdb := range joinableViewsDB {
		_, _exist := joinableViewsDBMerged[vdb.ID]
		if !_exist {
			joinableViewsDBMerged[vdb.ID] = []gensql.GetJoinableViewsForOwnerRow{}
		}
		joinableViewsDBMerged[vdb.ID] = append(joinableViewsDBMerged[vdb.ID], vdb)
	}

	joinableViews := []*JoinableView{}

	for k, v := range joinableViewsDBMerged {
		newJoinableView := JoinableView{
			ID:                k,
			Name:              v[0].Name,
			Created:           v[0].Created,
			Expires:           nullTimeToPtr(v[0].Expires),
			PseudoDatasources: []*models.BigQuery{},
		}
		joinableViews = append(joinableViews, &newJoinableView)
		for _, bq := range v {
			newJoinableView.PseudoDatasources = append(newJoinableView.PseudoDatasources, &models.BigQuery{
				ProjectID: bq.ProjectID,
				Dataset:   bq.DatasetID,
				Table:     bq.TableID,
			})
		}
	}
	return joinableViews, nil
}

func (r *Repo) SetJoinableViewDeleted(ctx context.Context, id uuid.UUID) error {
	return r.querier.SetJoinableViewDeleted(ctx, id)
}

func (r *Repo) MakeBigQueryUrlForJoinableViewDataset(name string) string {
	return fmt.Sprintf("%v.%v", r.centralDataProject, name)
}

func (r *Repo) GetJoinableViewsForReferenceAndUser(ctx context.Context, user string, pseudoDatasetID uuid.UUID) ([]gensql.GetJoinableViewsForReferenceAndUserRow, error) {
	joinableViews, err := r.querier.GetJoinableViewsForReferenceAndUser(ctx, gensql.GetJoinableViewsForReferenceAndUserParams{
		Owner:           user,
		PseudoDatasetID: pseudoDatasetID,
	})
	if err != nil {
		return nil, err
	}

	return joinableViews, nil
}

func (r *Repo) GetJoinableViewsWithReference(ctx context.Context) ([]gensql.GetJoinableViewsWithReferenceRow, error) {
	return r.querier.GetJoinableViewsWithReference(ctx)
}

func (r *Repo) GetJoinableViewsToBeDeletedWithRefDatasource(ctx context.Context) ([]gensql.GetJoinableViewsToBeDeletedWithRefDatasourceRow, error) {
	return r.querier.GetJoinableViewsToBeDeletedWithRefDatasource(ctx)
}

func (r *Repo) GetJoinableViewWithAccessStatus(ctx context.Context, joinableViewID uuid.UUID, user *auth.User) (*JoinableViewInDetail, error) {
	joinableViewDatasets, err := r.querier.GetJoinableViewWithDataset(ctx, joinableViewID)
	if err != nil {
		return nil, err
	}

	jvd := JoinableViewInDetail{
		JoinableView: JoinableView{
			ID:      joinableViewDatasets[0].JoinableViewID,
			Name:    joinableViewDatasets[0].JoinableViewName,
			Created: joinableViewDatasets[0].JoinableViewCreated,
			Expires: nullTimeToPtr(joinableViewDatasets[0].JoinableViewExpires),
		},
	}

OUTER:
	for _, jvds := range joinableViewDatasets {
		jvd.PseudoDatasources = append(jvd.PseudoDatasources, &models.BigQuery{
			ProjectID: jvds.BqProject,
			Dataset:   jvds.BqDataset,
			Table:     jvds.BqTable,
		})
		if user.GoogleGroups.Contains(jvds.Group) {
			jvd.AccessToViews = append(jvd.AccessToViews, true)
			continue
		}

		activeAccessList, err := r.ListActiveAccessToDataset(ctx, jvds.DatasetID)
		if err != nil {
			return nil, err
		}

		for _, access := range activeAccessList {
			if access.Subject == "user:"+user.Email {
				jvd.AccessToViews = append(jvd.AccessToViews, true)
				continue OUTER
			}
		}
		jvd.AccessToViews = append(jvd.AccessToViews, false)
	}

	return &jvd, nil
}
