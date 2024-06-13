package core

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"strings"
	"time"
)

type accessService struct {
	slackapi            service.SlackAPI
	pollyStorage        service.PollyStorage
	accessStorage       service.AccessStorage
	dataProductStorage  service.DataProductsStorage
	bigQueryStorage     service.BigQueryStorage
	joinableViewStorage service.JoinableViewsStorage
	bigQueryAPI         service.BigQueryAPI
}

func (s *accessService) GetAccessRequests(ctx context.Context, datasetID string) (*service.AccessRequestsWrapper, error) {
	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return nil, fmt.Errorf("parsing dataset id: %w", err)
	}

	requests, err := s.accessStorage.ListAccessRequestsForDataset(ctx, datasetUUID)
	if err != nil {
		return nil, fmt.Errorf("list access requests for dataset: %w", err)
	}

	for _, r := range requests {
		if r.Polly != nil {
			polly, err := s.pollyStorage.GetPollyDocumentation(ctx, r.Polly.ID)
			if err != nil {
				return nil, fmt.Errorf("get polly documentation: %w", err)
			}

			r.Polly = polly
		}
	}

	return &service.AccessRequestsWrapper{
		AccessRequests: requests,
	}, nil
}

func (s *accessService) CreateAccessRequest(ctx context.Context, input service.NewAccessRequestDTO) error {
	// FIXME: don't like this, lets see if we can do something about it
	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}

	owner := "user:" + user.Email
	if input.Owner != nil {
		owner = "group:" + *input.Owner
	}

	subjType := service.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType + ":" + subj

	var pollyID uuid.NullUUID
	if input.Polly != nil {
		dbPolly, err := s.pollyStorage.CreatePollyDocumentation(ctx, *input.Polly)
		if err != nil {
			return fmt.Errorf("create polly documentation: %w", err)
		}

		pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
	}

	accessRequest, err := s.accessStorage.CreateAccessRequestForDataset(ctx, input.DatasetID, pollyID, subjWithType, owner, input.Expires)
	if err != nil {
		return err
	}

	err = s.slackapi.InformNewAccessRequest(ctx, accessRequest.Owner, accessRequest.ID.String())
	if err != nil {
		return fmt.Errorf("inform new access request: %w", err)
	}

	return nil
}

func (s *accessService) DeleteAccessRequest(ctx context.Context, accessRequestID string) error {
	accessRequest, apierr := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	splits := strings.Split(accessRequest.Owner, ":")
	if len(splits) != 2 {
		return fmt.Errorf("owner is not in the correct format")
	}
	owner := splits[1]

	if err := ensureOwner(ctx, owner); err != nil {
		return fmt.Errorf("ensure owner: %w", err)
	}

	if err := s.accessStorage.DeleteAccessRequest(ctx, accessRequestID); err != nil {
		return fmt.Errorf("delete access request: %w", err)
	}

	return nil
}

func (s *accessService) UpdateAccessRequest(ctx context.Context, input service.UpdateAccessRequestDTO) error {
	if input.Polly != nil {
		if input.Polly.ID == nil {
			dbPolly, err := s.pollyStorage.CreatePollyDocumentation(ctx, *input.Polly)
			if err != nil {
				return fmt.Errorf("create polly documentation: %w", err)
			}

			input.Polly.ID = &dbPolly.ID
		}
	}

	err := s.accessStorage.UpdateAccessRequest(ctx, input)
	if err != nil {
		return fmt.Errorf("update access request: %w", err)
	}

	return nil
}

func (s *accessService) ApproveAccessRequest(ctx context.Context, accessRequestID string) error {
	ar, apierr := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	ds, apierr := s.dataProductStorage.GetDataset(ctx, ar.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	bq, apiError := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
	if apiError != nil {
		return apiError
	}

	dp, apiError := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apiError != nil {
		return apiError
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return fmt.Errorf("ensure user in group: %w", err)
	}

	if ds.Pii == "sensitive" && ar.Subject == "all-users@nav.no" {
		return fmt.Errorf("cannot approve access request for sensitive dataset for all-users")
	}

	subjWithType := ar.SubjectType + ":" + ar.Subject
	if err := s.bigQueryAPI.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return fmt.Errorf("grant access: %w", err)
	}

	err := s.accessStorage.GrantAccessToDatasetAndApproveRequest(
		ctx,
		ds.ID.String(),
		subjWithType,
		ar.ID.String(),
		ar.Expires,
	)
	if err != nil {
		return fmt.Errorf("grant access to dataset and approve request: %w", err)
	}

	return nil
}

func (s *accessService) DenyAccessRequest(ctx context.Context, accessRequestID string, reason *string) error {
	ar, apierr := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if apierr != nil {
		return apierr
	}

	ds, apierr := s.dataProductStorage.GetDataset(ctx, ar.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return apierr
	}

	err := ensureUserInGroup(ctx, dp.Owner.Group)
	if err != nil {
		return fmt.Errorf("ensure user in group: %w", err)
	}

	err = s.accessStorage.DenyAccessRequest(ctx, accessRequestID, reason)
	if err != nil {
		return fmt.Errorf("deny access request: %w", err)
	}

	return nil
}

func (s *accessService) RevokeAccessToDataset(ctx context.Context, id, gcpProjectID string) error {
	accessID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parsing access id: %w", err)
	}
	access, err := s.accessStorage.GetAccessToDataset(ctx, accessID)
	if err != nil {
		return fmt.Errorf("get access to dataset: %w", err)
	}

	ds, apierr := s.dataProductStorage.GetDataset(ctx, access.DatasetID.String())
	if apierr != nil {
		return apierr
	}

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return apierr
	}

	bqds, apierr := s.bigQueryStorage.GetBigqueryDatasource(ctx, access.DatasetID, false)
	if apierr != nil {
		return apierr
	}

	user := auth.GetUser(ctx)
	err = ensureUserInGroup(ctx, dp.Owner.Group)
	if err != nil && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return fmt.Errorf("ensure user in group: %w", err)
	}

	subjectParts := strings.Split(access.Subject, ":")
	if len(subjectParts) != 2 {
		return fmt.Errorf("subject is not in the correct format")
	}

	subjectWithoutType := subjectParts[1]

	if len(bqds.PseudoColumns) > 0 {
		joinableViews, err := s.joinableViewStorage.GetJoinableViewsForReferenceAndUser(ctx, subjectWithoutType, ds.ID)
		if err != nil {
			return fmt.Errorf("get joinable views for reference and user: %w", err)
		}
		for _, jv := range joinableViews {
			// FIXME: this is a bit of a hack, we should probably have a better way to get the joinable view name
			joinableViewName := makeJoinableViewName(bqds.ProjectID, bqds.Dataset, bqds.Table)
			if err := s.bigQueryAPI.Revoke(ctx, gcpProjectID, jv.Dataset, joinableViewName, access.Subject); err != nil {
				return fmt.Errorf("revoke access: %w", err)
			}
		}
	}

	if err := s.bigQueryAPI.Revoke(ctx, bqds.ProjectID, bqds.Dataset, bqds.Table, access.Subject); err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	if err := s.accessStorage.RevokeAccessToDataset(ctx, accessID); err != nil {
		return fmt.Errorf("revoke access to dataset: %w", err)
	}

	return nil
}

// FIXME: duplicated
func makeJoinableViewName(projectID, datasetID, tableID string) string {
	// datasetID will always be same markedsplassen dataset id
	return fmt.Sprintf("%v_%v", projectID, tableID)
}

func (s *accessService) GrantAccessToDataset(ctx context.Context, input service.GrantAccessData, gcpProjectID string) error {
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return fmt.Errorf("datoen tilgangen skal utløpe må være fram i tid")
	}

	user := auth.GetUser(ctx)
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}
	ds, apierr := s.dataProductStorage.GetDataset(ctx, input.DatasetID.String())
	if apierr != nil {
		return fmt.Errorf("get dataset: %w", apierr)
	}

	dp, apierr := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID.String())
	if apierr != nil {
		return fmt.Errorf("get dataproduct: %w", apierr)
	}

	if err := ensureUserInGroup(ctx, dp.Owner.Group); err != nil {
		return fmt.Errorf("ensure user in group: %w", err)
	}

	if ds.Pii == "sensitive" && subj == "all-users@nav.no" {
		return fmt.Errorf("datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere")
	}

	bqds, apierr := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
	if apierr != nil {
		return fmt.Errorf("get bigquery datasource: %w", apierr)
	}

	subjType := service.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}

	subjWithType := subjType + ":" + subj

	if len(bqds.PseudoColumns) > 0 {
		joinableViews, err := s.joinableViewStorage.GetJoinableViewsForReferenceAndUser(ctx, subj, ds.ID)
		if err != nil {
			return fmt.Errorf("get joinable views for reference and user: %w", err)
		}
		for _, jv := range joinableViews {
			joinableViewName := makeJoinableViewName(bqds.ProjectID, bqds.Dataset, bqds.Table)
			if err := s.bigQueryAPI.Grant(ctx, gcpProjectID, jv.Dataset, joinableViewName, subjWithType); err != nil {
				return fmt.Errorf("grant access: %w", err)
			}
		}
	}

	if err := s.bigQueryAPI.Grant(ctx, bqds.ProjectID, bqds.Dataset, bqds.Table, subjWithType); err != nil {
		return fmt.Errorf("grant access: %w", err)
	}

	err := s.accessStorage.GrantAccessToDatasetAndRenew(ctx, input.DatasetID, input.Expires, subjWithType, user.Email)
	if err != nil {
		return fmt.Errorf("grant access to dataset and renew: %w", err)
	}

	return nil
}

// FIXME: still dont like this
func ensureOwner(ctx context.Context, owner string) error {
	user := auth.GetUser(ctx)

	if user != nil && (user.GoogleGroups.Contains(owner) || owner == user.Email) {
		return nil
	}

	return service.ErrUnauthorized
}

func NewAccessService(slackapi service.SlackAPI, pollyStorage service.PollyStorage, accessStorage service.AccessStorage) *accessService {
	return &accessService{
		slackapi:      slackapi,
		pollyStorage:  pollyStorage,
		accessStorage: accessStorage,
	}
}
