package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type JoinableView struct {
	ID      uuid.UUID
	Name    string
	Created time.Time
	Expires *time.Time
}

type JoinableViewsDatasource struct {
	ProjectID  string
	DatasetID  string
	TableID    string
	Accessible bool
	Deleted    bool
}

type JoinableViewWithDatasource struct {
	JoinableView
	PseudoDatasources []*JoinableViewsDatasource
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
			ID:      k,
			Name:    v[0].Name,
			Created: v[0].Created,
			Expires: nullTimeToPtr(v[0].Expires),
		}
		joinableViews = append(joinableViews, &newJoinableView)
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

func (r *Repo) GetJoinableViewWithDatasource(ctx context.Context, joinableViewID uuid.UUID, user *auth.User) (*JoinableViewWithDatasource, error) {
	joinableViewDatasets, err := r.querier.GetJoinableViewWithDataset(ctx, joinableViewID)
	if err != nil {
		return nil, err
	}

	jvd := JoinableViewWithDatasource{
		JoinableView: JoinableView{
			ID:      joinableViewDatasets[0].JoinableViewID,
			Name:    joinableViewDatasets[0].JoinableViewName,
			Created: joinableViewDatasets[0].JoinableViewCreated,
			Expires: nullTimeToPtr(joinableViewDatasets[0].JoinableViewExpires),
		},
	}

	for _, ds := range joinableViewDatasets {
		jvbq := &JoinableViewsDatasource{
			ProjectID: ds.BqProject,
			DatasetID: ds.BqDataset,
			TableID:   ds.BqTable,
			Deleted:   ds.Deleted.Valid,
		}
		jvd.PseudoDatasources = append(jvd.PseudoDatasources, jvbq)

		if jvbq.Deleted {
			continue
		}

		if user.GoogleGroups.Contains(ds.Group.String) {
			jvbq.Accessible = true
		} else {
			activeAccessList, err := r.ListActiveAccessToDataset(ctx, ds.DatasetID.UUID)
			if err != nil {
				return nil, err
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

	return &jvd, nil
}
