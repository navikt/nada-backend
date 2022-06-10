package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *mutationResolver) GrantAccessToDataset(ctx context.Context, input models.NewGrant) (*models.Access, error) {
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return nil, fmt.Errorf("expires has already expired")
	}

	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}
	ds, err := r.repo.GetDataset(ctx, input.DatasetID)
	if err != nil {
		return nil, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return nil, err
	}
	if err := isAllowedToGrantAccess(ctx, r.repo, dp, ds.ID, subj, user); err != nil {
		return nil, err
	}

	if ds.Pii && subj == "all-users@nav.no" {
		return nil, fmt.Errorf("not allowed to grant all-users access to dataproduct containing pii")
	}

	bq, err := r.repo.GetBigqueryDatasource(ctx, ds.ID)
	if err != nil {
		return nil, err
	}

	subjType := models.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType.String() + ":" + subj

	if err := r.accessMgr.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return nil, err
	}

	return r.repo.GrantAccessToDataset(ctx, input.DatasetID, input.Expires, subjWithType, user.Email)
}

func (r *mutationResolver) RevokeAccessToDataset(ctx context.Context, id uuid.UUID) (bool, error) {
	access, err := r.repo.GetAccessToDataset(ctx, id)
	if err != nil {
		return false, err
	}

	ds, err := r.repo.GetDataset(ctx, access.DatasetID)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return false, err
	}

	bq, err := r.repo.GetBigqueryDatasource(ctx, access.DatasetID)
	if err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if !user.Groups.Contains(dp.Owner.Group) && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return false, ErrUnauthorized
	}

	if err := r.accessMgr.Revoke(ctx, bq.ProjectID, bq.Dataset, bq.Table, access.Subject); err != nil {
		return false, err
	}
	return true, r.repo.RevokeAccessToDataset(ctx, id)
}

func (r *mutationResolver) CreateAccessRequest(ctx context.Context, input models.NewAccessRequest) (*models.AccessRequest, error) {
	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}

	owner := "user:" + user.Email
	if input.Owner != nil {
		owner = "group:" + *input.Owner
	}

	subjType := models.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType.String() + ":" + subj

	var pollyID uuid.NullUUID
	if input.Polly != nil {
		dbPolly, err := r.repo.CreatePollyDocumentation(ctx, *input.Polly)
		if err != nil {
			return nil, err
		}

		pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
	}

	return r.repo.CreateAccessRequestForDataset(ctx, input.DatasetID, pollyID, subjWithType, owner, input.Expires)
}

func (r *mutationResolver) UpdateAccessRequest(ctx context.Context, input models.UpdateAccessRequest) (*models.AccessRequest, error) {
	var pollyID uuid.NullUUID
	if input.Polly != nil {
		if input.Polly.ID != nil {
			// Keep existing polly
			pollyID = uuid.NullUUID{UUID: *input.Polly.ID, Valid: true}
		} else {
			dbPolly, err := r.repo.CreatePollyDocumentation(ctx, *input.Polly)
			if err != nil {
				return nil, err
			}
			pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
		}
	}

	return r.repo.UpdateAccessRequest(ctx, input.ID, pollyID, input.Owner, input.Expires)
}

func (r *mutationResolver) DeleteAccessRequest(ctx context.Context, id uuid.UUID) (bool, error) {
	accessRequest, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	splits := strings.Split(accessRequest.Owner, ":")
	if len(splits) != 2 {
		return false, fmt.Errorf("%v is not a valid owner format (cannot split on :)", accessRequest.Owner)
	}
	owner := splits[1]

	if err := ensureOwner(ctx, owner); err != nil {
		return false, err
	}

	if err := r.repo.DeleteAccessRequest(ctx, id); err != nil {
		return false, err
	}

	return true, nil
}

func (r *mutationResolver) ApproveAccessRequest(ctx context.Context, id uuid.UUID) (bool, error) {
	ar, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	ds, err := r.repo.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return false, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if err := r.repo.ApproveAccessRequest(ctx, id, user.Email); err != nil {
		return false, err
	}

	return true, nil
}

func (r *mutationResolver) DenyAccessRequest(ctx context.Context, id uuid.UUID, reason *string) (bool, error) {
	ar, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	ds, err := r.repo.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return false, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if err := r.repo.DenyAccessRequest(ctx, id, user.Email, reason); err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) AccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	return r.repo.GetAccessRequest(ctx, id)
}