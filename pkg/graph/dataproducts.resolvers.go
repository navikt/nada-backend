package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *dataproductResolver) Datasource(ctx context.Context, obj *models.Dataproduct) (models.Datasource, error) {
	return r.repo.GetBigqueryDatasource(ctx, obj.ID)
}

func (r *dataproductResolver) Requesters(ctx context.Context, obj *models.Dataproduct) ([]string, error) {
	return r.repo.GetDataproductRequesters(ctx, obj.ID)
}

func (r *dataproductResolver) Access(ctx context.Context, obj *models.Dataproduct) ([]*models.Access, error) {
	if err := ensureUserInGroup(ctx, obj.Owner.Group); err != nil {
		return nil, err
	}
	return r.repo.ListAccessToDataproduct(ctx, obj.ID)
}

func (r *mutationResolver) CreateDataproduct(ctx context.Context, input models.NewDataproduct) (*models.Dataproduct, error) {
	if err := ensureUserInGroup(ctx, input.Group); err != nil {
		return nil, err
	}
	return r.repo.CreateDataproduct(ctx, input)
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
	if err := r.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, subjType.String()+":"+subj); err != nil {
		return nil, err
	}

	return r.repo.GrantAccessToDataproduct(ctx, dataproductID, expires, subj, user.Email)
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

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err == nil {
		return true, r.repo.RevokeAccessToDataproduct(ctx, id)
	}

	user := auth.GetUser(ctx)
	if user.Email == access.Subject {
		return true, r.repo.RevokeAccessToDataproduct(ctx, id)
	}

	return false, ErrUnauthorized
}

func (r *queryResolver) Dataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, id)
}

func (r *queryResolver) Dataproducts(ctx context.Context, limit *int, offset *int) ([]*models.Dataproduct, error) {
	l, o := pagination(limit, offset)
	return r.repo.GetDataproducts(ctx, l, o)
}

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

type dataproductResolver struct{ *Resolver }
