package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/graph/generated"
	"github.com/navikt/nada-backend/pkg/graph/models"
	log "github.com/sirupsen/logrus"
)

func (r *bigQueryResolver) Schema(ctx context.Context, obj *models.BigQuery) ([]*models.TableColumn, error) {
	return r.repo.GetDataproductMetadata(ctx, obj.DataproductID)
}

func (r *dataproductResolver) Datasource(ctx context.Context, obj *models.Dataproduct) (models.Datasource, error) {
	return r.repo.GetBigqueryDatasource(ctx, obj.ID)
}

func (r *dataproductResolver) Requesters(ctx context.Context, obj *models.Dataproduct) ([]string, error) {
	allRequesters, err := r.repo.GetDataproductRequesters(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)
	if user.Groups.Contains(obj.Owner.Group) {
		return allRequesters, nil
	}

	var ret []string
	for _, r := range allRequesters {
		if strings.EqualFold(r, user.Email) {
			ret = append(ret, r)
		} else if user.Groups.Contains(r) {
			ret = append(ret, r)
		}
	}

	return ret, nil
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

	var ret []*models.Access
	for _, a := range all {
		if strings.EqualFold(a.Subject, "user:"+user.Email) {
			ret = append(ret, a)
		} else if strings.HasPrefix(a.Subject, "group:") && user.Groups.Contains(strings.TrimPrefix(a.Subject, "group:")) {
			ret = append(ret, a)
		}
	}

	return ret, nil
}

func (r *dataproductResolver) Services(ctx context.Context, obj *models.Dataproduct) (*models.DataproductServices, error) {
	meta, err := r.repo.GetMetabaseMetadata(ctx, obj.ID, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.DataproductServices{}, nil
		}
		return nil, err
	}

	svc := &models.DataproductServices{}
	if meta.DatabaseID > 0 {
		base := "https://metabase.dev.intern.nav.no/browse/%v"
		if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
			base = "https://metabase.intern.nav.no/browse/%v"
		}
		url := fmt.Sprintf(base, meta.DatabaseID)
		svc.Metabase = &url
	}

	return svc, nil
}

func (r *dataproductResolver) Mappings(ctx context.Context, obj *models.Dataproduct) ([]models.MappingService, error) {
	return r.repo.GetDataproductMappings(ctx, obj.ID)
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
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
	}
	dp, err := r.repo.CreateDataproduct(ctx, input)
	if err != nil {
		return nil, err
	}
	err = r.slack.NewDataproduct(dp)
	if err != nil {
		log.Errorf("failed to send slack notification: %v", err)
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
	if input.Description != nil && *input.Description != "" {
		*input.Description = html.EscapeString(*input.Description)
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

func (r *mutationResolver) GrantAccessToDataproduct(ctx context.Context, input models.NewGrant) (*models.Access, error) {
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return nil, fmt.Errorf("expires has already expired")
	}

	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}
	dp, err := r.repo.GetDataproduct(ctx, input.DataproductID)
	if err != nil {
		return nil, err
	}
	if err := isAllowedToGrantAccess(ctx, r.repo, dp, subj, user); err != nil {
		return nil, err
	}

	if dp.Pii && subj == "all-users@nav.no" {
		return nil, fmt.Errorf("not allowed to grant all-users access to dataproduct containing pii")
	}

	ds, err := r.repo.GetBigqueryDatasource(ctx, dp.ID)
	if err != nil {
		return nil, err
	}

	subjType := models.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType.String() + ":" + subj

	if err := r.accessMgr.Grant(ctx, ds.ProjectID, ds.Dataset, ds.Table, subjWithType); err != nil {
		return nil, err
	}

	return r.repo.GrantAccessToDataproduct(ctx, input.DataproductID, input.Expires, subjWithType, user.Email)
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
	if !user.Groups.Contains(dp.Owner.Group) && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return false, ErrUnauthorized
	}

	if err := r.accessMgr.Revoke(ctx, ds.ProjectID, ds.Dataset, ds.Table, access.Subject); err != nil {
		return false, err
	}
	return true, r.repo.RevokeAccessToDataproduct(ctx, id)
}

func (r *mutationResolver) MapDataproduct(ctx context.Context, dataproductID uuid.UUID, services []models.MappingService) (bool, error) {
	dp, err := r.repo.GetDataproduct(ctx, dataproductID)
	if err != nil {
		return false, err
	}
	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	err = r.repo.MapDataproduct(ctx, dataproductID, services)
	if err != nil {
		return false, err
	}
	return true, nil
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

	return r.repo.CreateAccessRequestForDataproduct(ctx, input.DataproductID, pollyID, subjWithType, owner)
}

func (r *mutationResolver) UpdateAccessRequest(ctx context.Context, input models.UpdateAccessRequest) (*models.AccessRequest, error) {
	var pollyID uuid.NullUUID
	if input.NewPolly != nil {
		dbPolly, err := r.repo.CreatePollyDocumentation(ctx, *input.NewPolly)
		if err != nil {
			return nil, err
		}
		pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
	} else if input.PollyID != nil {
		pollyID = uuid.NullUUID{UUID: *input.PollyID, Valid: true}
	}

	return r.repo.UpdateAccessRequest(ctx, input.ID, pollyID, input.Owner)
}

func (r *mutationResolver) DeleteAccessRequest(ctx context.Context, id uuid.UUID) (bool, error) {
	accessRequest, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	splits := strings.Split(*accessRequest.Owner, ":")
	if len(splits) != 2 {
		return false, fmt.Errorf("%v is not a valid owner format (cannot split on :)", *accessRequest.Owner)
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

	dp, err := r.repo.GetDataproduct(ctx, ar.DataproductID)
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

func (r *mutationResolver) DenyAccessRequest(ctx context.Context, id uuid.UUID) (bool, error) {
	ar, err := r.repo.GetAccessRequest(ctx, id)
	if err != nil {
		return false, err
	}

	dp, err := r.repo.GetDataproduct(ctx, ar.DataproductID)
	if err != nil {
		return false, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return false, err
	}

	user := auth.GetUser(ctx)
	if err := r.repo.DenyAccessRequest(ctx, id, user.Email); err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) Dataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	return r.repo.GetDataproduct(ctx, id)
}

func (r *queryResolver) Dataproducts(ctx context.Context, limit *int, offset *int, service *models.MappingService) ([]*models.Dataproduct, error) {
	l, o := pagination(limit, offset)
	if service != nil {
		switch *service {
		case models.MappingServiceMetabase:
			return r.repo.GetDataproductsByMetabase(ctx, l, o)
		default:
			return nil, fmt.Errorf("unknown service: %s", *service)
		}
	}
	return r.repo.GetDataproducts(ctx, l, o)
}

func (r *queryResolver) GroupStats(ctx context.Context, limit *int, offset *int) ([]*models.GroupStats, error) {
	l, o := pagination(limit, offset)
	return r.repo.DataproductGroupStats(ctx, l, o)
}

func (r *queryResolver) AccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	return r.repo.GetAccessRequest(ctx, id)
}

func (r *queryResolver) AccessRequestsForDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.AccessRequest, error) {
	dp, err := r.repo.GetDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return nil, err
	}

	return r.repo.ListAccessRequestsForDataproduct(ctx, dataproductID)
}

// BigQuery returns generated.BigQueryResolver implementation.
func (r *Resolver) BigQuery() generated.BigQueryResolver { return &bigQueryResolver{r} }

// Dataproduct returns generated.DataproductResolver implementation.
func (r *Resolver) Dataproduct() generated.DataproductResolver { return &dataproductResolver{r} }

type bigQueryResolver struct{ *Resolver }
type dataproductResolver struct{ *Resolver }
