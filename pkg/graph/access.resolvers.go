package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.41

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/bigquery"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

// AccessRequest is the resolver for the accessRequest field.
func (r *accessResolver) AccessRequest(ctx context.Context, obj *models.Access) (*models.AccessRequest, error) {
	if obj.AccessRequestID == nil {
		return nil, nil
	}
	return r.repo.GetAccessRequest(ctx, *obj.AccessRequestID)
}

// GrantAccessToDataset is the resolver for the grantAccessToDataset field.
func (r *mutationResolver) GrantAccessToDataset(ctx context.Context, input models.NewGrant) (*models.Access, error) {
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return nil, fmt.Errorf("Datoen tilgangen skal utløpe må være fram i tid.")
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
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	if ds.Pii == "sensitive" && subj == "all-users@nav.no" {
		return nil, fmt.Errorf("Datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere (all-users@nav.no).")
	}

	bq, err := r.repo.GetBigqueryDatasource(ctx, ds.ID, false)
	if err != nil {
		return nil, err
	}

	subjType := models.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType.String() + ":" + subj

	if len(bq.PseudoColumns) > 0 {
		joinableViews, err := r.repo.GetJoinableViewsForReferenceAndUser(ctx, subj, ds.ID)
		if err != nil {
			return nil, err
		}
		for _, jv := range joinableViews {
			joinableViewName := bigquery.MakeJoinableViewName(bq.ProjectID, bq.Dataset, bq.Table)
			if err := r.accessMgr.Grant(ctx, r.centralDataProject, jv.Dataset, joinableViewName, subjWithType); err != nil {
				return nil, err
			}
		}
	}

	if err := r.accessMgr.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return nil, err
	}

	return r.repo.GrantAccessToDataset(ctx, input.DatasetID, input.Expires, subjWithType, user.Email)
}

// RevokeAccessToDataset is the resolver for the revokeAccessToDataset field.
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

	bq, err := r.repo.GetBigqueryDatasource(ctx, access.DatasetID, false)
	if err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(dp.Owner.Group) && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return false, ErrUnauthorized
	}

	subjectParts := strings.Split(access.Subject, ":")
	if len(subjectParts) != 2 {
		return false, fmt.Errorf("invalid access subject %v (should be on format type:email)", access.Subject)
	}
	subjectWithoutType := subjectParts[1]

	if len(bq.PseudoColumns) > 0 {
		joinableViews, err := r.repo.GetJoinableViewsForReferenceAndUser(ctx, subjectWithoutType, ds.ID)
		if err != nil {
			return false, err
		}
		for _, jv := range joinableViews {
			joinableViewName := bigquery.MakeJoinableViewName(bq.ProjectID, bq.Dataset, bq.Table)
			if err := r.accessMgr.Revoke(ctx, r.centralDataProject, jv.Dataset, joinableViewName, access.Subject); err != nil {
				return false, err
			}
		}
	}

	if err := r.accessMgr.Revoke(ctx, bq.ProjectID, bq.Dataset, bq.Table, access.Subject); err != nil {
		return false, err
	}
	return true, r.repo.RevokeAccessToDataset(ctx, id)
}

// CreateAccessRequest is the resolver for the createAccessRequest field.
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

	ar, err := r.repo.CreateAccessRequestForDataset(ctx, input.DatasetID, pollyID, subjWithType, owner, input.Expires)
	if err != nil {
		return nil, err
	}
	r.SendNewAccessRequestSlackNotification(ctx, ar)
	return ar, nil
}

// UpdateAccessRequest is the resolver for the updateAccessRequest field.
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

// DeleteAccessRequest is the resolver for the deleteAccessRequest field.
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

// ApproveAccessRequest is the resolver for the approveAccessRequest field.
func (r *mutationResolver) ApproveAccessRequest(ctx context.Context, id uuid.UUID) (bool, error) {
	ar, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	ds, err := r.repo.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		return false, err
	}

	bq, err := r.repo.GetBigqueryDatasource(ctx, ds.ID, false)
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

	if ds.Pii == "sensitive" && ar.Subject == "all-users@nav.no" {
		return false, fmt.Errorf("Datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere (all-users@nav.no).")
	}

	subjWithType := ar.SubjectType.String() + ":" + ar.Subject
	if err := r.accessMgr.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if err := r.repo.ApproveAccessRequest(ctx, id, user.Email); err != nil {
		return false, err
	}

	return true, nil
}

// DenyAccessRequest is the resolver for the denyAccessRequest field.
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

// AccessRequest is the resolver for the accessRequest field.
func (r *queryResolver) AccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	return r.repo.GetAccessRequest(ctx, id)
}

// Access returns generated.AccessResolver implementation.
func (r *Resolver) Access() generated.AccessResolver { return &accessResolver{r} }

type accessResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *mutationResolver) SendNewAccessRequestSlackNotification(ctx context.Context, ar *models.AccessRequest) {
	ds, err := r.repo.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		r.log.Warn("Access request created but failed to fetch dataset during sending slack notification", err)
		return
	}

	dp, err := r.repo.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		r.log.Warn("Access request created but failed to fetch dataproduct during sending slack notification", err)
		return
	}

	if dp.Owner.TeamContact == nil || *dp.Owner.TeamContact == "" {
		r.log.Info("Access request created but skip slack message because teamcontact is empty")
		return
	}

	err = r.slack.NewAccessRequest(*dp.Owner.TeamContact, dp, ds, ar)
	if err != nil {
		r.log.Warn("Access request created, failed to send slack message", err)
	}
}
