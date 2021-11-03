package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *bigQueryResolver) Schema(ctx context.Context, obj *models.BigQuery) ([]*models.TableColumn, error) {
	return r.repo.GetDataproductMetadata(ctx, obj.DataproductID)
}

func (r *dataproductResolver) Datasource(ctx context.Context, obj *models.Dataproduct) (models.Datasource, error) {
	return r.repo.GetBigqueryDatasource(ctx, obj.ID)
}

func (r *dataproductResolver) Requesters(ctx context.Context, obj *models.Dataproduct) ([]string, error) {
	return r.repo.GetDataproductRequesters(ctx, obj.ID)
}

func (r *dataproductResolver) Access(ctx context.Context, obj *models.Dataproduct) ([]*models.Access, error) {
	all, err := r.repo.ListAccessToDataproduct(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if user.Groups.Contains(obj.Owner.Group) {
		return all, nil
	}

	ret := []*models.Access{}
	for _, a := range all {
		if a.Subject == "user:"+user.Email {
			ret = append(ret, a)
		} else if strings.HasPrefix(a.Subject, "group:") && user.Groups.Contains(strings.TrimPrefix(a.Subject, "group:")) {
			ret = append(ret, a)
		}
	}

	return ret, nil
}

func (r *mutationResolver) CreateDataproduct(ctx context.Context, input models.NewDataproduct) (*models.Dataproduct, error) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, err
	}

	if err := r.ensureUserHasAccessToGcpProject(ctx, input.BigQuery.ProjectID); err != nil {
		return nil, err
	}

	metadata, err := r.bigquery.TableMetadata(ctx, input.BigQuery.ProjectID, input.BigQuery.Dataset, input.BigQuery.Table)
	if err != nil {
		return nil, fmt.Errorf("trying to create table %v, but it does not exist in %v.%v",
			input.BigQuery.Table, input.BigQuery.ProjectID, input.BigQuery.Dataset)
	}

	switch metadata.TableType {
	case bigquery.RegularTable:
	case bigquery.ViewTable:
		if err := r.accessMgr.AddToAuthorizedViews(ctx, input.BigQuery.ProjectID, input.BigQuery.Dataset, input.BigQuery.Table); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported table type: %v", metadata.TableType)
	}

	input.Metadata = metadata
	dp, err := r.repo.CreateDataproduct(ctx, input)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func (r *mutationResolver) UpdateDataproduct(ctx context.Context, id uuid.UUID, input models.UpdateDataproduct) (*models.Dataproduct, error) {
	dp, err := r.repo.GetDataproduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}
	return r.repo.UpdateDataproduct(ctx, id, input)
}

func (r *mutationResolver) DeleteDataproduct(ctx context.Context, id uuid.UUID) (bool, error) {
	dp, err := r.repo.GetDataproduct(ctx, id)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	return true, r.repo.DeleteDataproduct(ctx, dp.ID)
}

func (r *mutationResolver) AddRequesterToDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) (bool, error) {
	dp, err := r.repo.GetDataproduct(ctx, dataproductID)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	return true, r.repo.AddRequesterToDataproduct(ctx, dp.ID, subject)
}

func (r *mutationResolver) RemoveRequesterFromDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) (bool, error) {
	dp, err := r.repo.GetDataproduct(ctx, dataproductID)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	return true, r.repo.RemoveRequesterFromDataproduct(ctx, dp.ID, subject)
}

func (r *mutationResolver) GrantAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID, expires *time.Time, subject *string, subjectType *models.SubjectType) (*models.Access, error) {
	if expires != nil && expires.Before(time.Now()) {
		return nil, fmt.Errorf("expires has already expired")
	}

	user := auth.GetUser(ctx)
	subj := user.Email
	if subject != nil {
		subj = *subject
	}
	dp, err := r.repo.GetDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}
	if err := isAllowedToGrantAccess(ctx, r.repo, dp, subj, user); err != nil {
		return nil, err
	}

	ds, err := r.repo.GetBigqueryDatasource(ctx, dp.ID)
	if err != nil {
		return nil, err
	}

	subjType := models.SubjectTypeUser
	if subjectType != nil {
		subjType = *subjectType
	}

	subjWithType := subjType.String() + ":" + subj

	if err := r.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, subjWithType); err != nil {
		return nil, err
	}

	return r.repo.GrantAccessToDataproduct(ctx, dataproductID, expires, subjWithType, user.Email)
}

func (r *mutationResolver) RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) (bool, error) {
	access, err := r.repo.GetAccessToDataproduct(ctx, id)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, access.DataproductID)
	if err != nil {
		return false, err
	}

	ds, err := r.repo.GetBigqueryDatasource(ctx, access.DataproductID)
	if err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(dp.Owner.Group) && "user:"+user.Email != access.Subject {
		return false, ErrUnauthorized
	}

	if err := r.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, access.Subject); err != nil {
		return false, err
	}
	return true, r.repo.RevokeAccessToDataproduct(ctx, id)
}

func (r *queryResolver) Dataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, id)
}

func (r *queryResolver) Dataproducts(ctx context.Context, limit *int, offset *int) ([]*models.Dataproduct, error) {
	l, o := pagination(limit, offset)
	return r.repo.GetDataproducts(ctx, l, o)
}

// BigQuery returns generated.BigQueryResolver implementation.
func (r *Resolver) BigQuery() generated.BigQueryResolver { return &bigQueryResolver{r} }

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

type bigQueryResolver struct{ *Resolver }
type dataproductResolver struct{ *Resolver }
