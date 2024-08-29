package core

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

var _ service.AccessService = (*accessService)(nil)

type accessService struct {
	dataCatalogueURL    string
	slackapi            service.SlackAPI
	pollyStorage        service.PollyStorage
	accessStorage       service.AccessStorage
	dataProductStorage  service.DataProductsStorage
	bigQueryStorage     service.BigQueryStorage
	joinableViewStorage service.JoinableViewsStorage
	bigQueryAPI         service.BigQueryAPI
}

func (s *accessService) GetAccessRequests(ctx context.Context, datasetID uuid.UUID) (*service.AccessRequestsWrapper, error) {
	const op errs.Op = "accessService.GetAccessRequests"

	requests, err := s.accessStorage.ListAccessRequestsForDataset(ctx, datasetID)
	if err != nil {
		return nil, errs.E(op, err)
	}

	for _, r := range requests {
		if r.Polly != nil {
			polly, err := s.pollyStorage.GetPollyDocumentation(ctx, r.Polly.ID)
			if err != nil {
				return nil, errs.E(op, err)
			}

			r.Polly = polly
		}
	}

	return &service.AccessRequestsWrapper{
		AccessRequests: requests,
	}, nil
}

func (s *accessService) CreateAccessRequest(ctx context.Context, user *service.User, input service.NewAccessRequestDTO) error {
	const op errs.Op = "accessService.CreateAccessRequest"

	// FIXME: Can we stop with the joining and splitting of user, group, email, etc.
	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}

	subjType := service.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}
	subjWithType := subjType + ":" + subj

	owner := subjWithType
	if subjType == service.SubjectTypeServiceAccount {
		if input.Owner != nil {
			owner = service.SubjectTypeGroup + ":" + *input.Owner
		} else {
			owner = service.SubjectTypeUser + ":" + user.Email
		}
	}

	var pollyID uuid.NullUUID
	if input.Polly != nil {
		dbPolly, err := s.pollyStorage.CreatePollyDocumentation(ctx, *input.Polly)
		if err != nil {
			return errs.E(op, err)
		}

		pollyID = uuid.NullUUID{UUID: dbPolly.ID, Valid: true}
	}

	accessRequest, err := s.accessStorage.CreateAccessRequestForDataset(ctx, input.DatasetID, pollyID, subjWithType, owner, input.Expires)
	if err != nil {
		return errs.E(op, err)
	}

	ds, err := s.dataProductStorage.GetDataset(ctx, input.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	slackMessage := createAccessRequestSlackNotification(dp, ds, s.dataCatalogueURL, accessRequest.Owner)

	if dp.Owner.TeamContact == nil || *dp.Owner.TeamContact == "" {
		return nil
	}

	err = s.slackapi.SendSlackNotification(*dp.Owner.TeamContact, slackMessage)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func createAccessRequestSlackNotification(dp *service.DataproductWithDataset, ds *service.Dataset, dataCatalogueURL, subject string) string {
	link := fmt.Sprintf(
		"\nLink: %s/dataproduct/%s/%s/%s",
		dataCatalogueURL,
		dp.ID.String(),
		url.QueryEscape(dp.Name),
		ds.ID.String(),
	)

	dsp := fmt.Sprintf(
		"\nDatasett: %s\nDataprodukt: %s",
		ds.Name,
		dp.Name,
	)

	return fmt.Sprintf(
		"%s har sendt en søknad om tilgang for: %s%s",
		subject,
		dsp,
		link,
	)
}

func (s *accessService) DeleteAccessRequest(ctx context.Context, user *service.User, accessRequestID uuid.UUID) error {
	const op errs.Op = "accessService.DeleteAccessRequest"

	accessRequest, err := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		return errs.E(op, err)
	}

	if err := ensureOwner(user, accessRequest.Owner); err != nil {
		return errs.E(op, err)
	}

	if err := s.accessStorage.DeleteAccessRequest(ctx, accessRequestID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *accessService) UpdateAccessRequest(ctx context.Context, input service.UpdateAccessRequestDTO) error {
	const op errs.Op = "accessService.UpdateAccessRequest"

	// FIXME: Should we allow updating without checking the owner?

	if input.Polly != nil {
		if input.Polly.ID == nil {
			dbPolly, err := s.pollyStorage.CreatePollyDocumentation(ctx, *input.Polly)
			if err != nil {
				return errs.E(op, err)
			}

			input.Polly.ID = &dbPolly.ID
		}
	}

	err := s.accessStorage.UpdateAccessRequest(ctx, input)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *accessService) ApproveAccessRequest(ctx context.Context, user *service.User, accessRequestID uuid.UUID) error {
	const op errs.Op = "accessService.ApproveAccessRequest"

	ar, err := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		return errs.E(op, err)
	}

	ds, err := s.dataProductStorage.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	bq, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	err = ensureUserInGroup(user, dp.Owner.Group)
	if err != nil {
		return errs.E(op, err)
	}

	if ds.Pii == "sensitive" && ar.Subject == "all-users@nav.no" {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere"))
	}

	subjWithType := ar.SubjectType + ":" + ar.Subject
	if err := s.bigQueryAPI.Grant(ctx, bq.ProjectID, bq.Dataset, bq.Table, subjWithType); err != nil {
		return errs.E(op, err)
	}

	err = s.accessStorage.GrantAccessToDatasetAndApproveRequest(
		ctx,
		user,
		ds.ID,
		subjWithType,
		ar.Owner,
		ar.ID,
		ar.Expires,
	)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *accessService) DenyAccessRequest(ctx context.Context, user *service.User, accessRequestID uuid.UUID, reason *string) error {
	const op errs.Op = "accessService.DenyAccessRequest"

	ar, err := s.accessStorage.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		return errs.E(op, err)
	}

	ds, err := s.dataProductStorage.GetDataset(ctx, ar.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	err = ensureUserInGroup(user, dp.Owner.Group)
	if err != nil {
		return errs.E(op, err)
	}

	err = s.accessStorage.DenyAccessRequest(ctx, user, accessRequestID, reason)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func (s *accessService) RevokeAccessToDataset(ctx context.Context, user *service.User, accessID uuid.UUID, gcpProjectID string) error {
	const op errs.Op = "accessService.RevokeAccessToDataset"

	access, err := s.accessStorage.GetAccessToDataset(ctx, accessID)
	if err != nil {
		return errs.E(op, err)
	}

	ds, err := s.dataProductStorage.GetDataset(ctx, access.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	bqds, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, access.DatasetID, false)
	if err != nil {
		return errs.E(op, err)
	}

	err = ensureUserInGroup(user, dp.Owner.Group)
	if err != nil && !strings.EqualFold("user:"+user.Email, access.Subject) {
		return errs.E(op, err)
	}

	subjectParts := strings.Split(access.Subject, ":")
	if len(subjectParts) != 2 {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("subject is not in the correct format"))
	}

	subjectWithoutType := subjectParts[1]

	if len(bqds.PseudoColumns) > 0 {
		joinableViews, err := s.joinableViewStorage.GetJoinableViewsForReferenceAndUser(ctx, subjectWithoutType, ds.ID)
		if err != nil {
			return errs.E(op, err)
		}

		for _, jv := range joinableViews {
			// FIXME: this is a bit of a hack, we should probably have a better way to get the joinable view name
			joinableViewName := makeJoinableViewName(bqds.ProjectID, bqds.Dataset, bqds.Table)
			if err := s.bigQueryAPI.Revoke(ctx, gcpProjectID, jv.Dataset, joinableViewName, access.Subject); err != nil {
				return errs.E(op, err)
			}
		}
	}

	if err := s.bigQueryAPI.Revoke(ctx, bqds.ProjectID, bqds.Dataset, bqds.Table, access.Subject); err != nil {
		return errs.E(op, err)
	}

	if err := s.accessStorage.RevokeAccessToDataset(ctx, accessID); err != nil {
		return errs.E(op, err)
	}

	return nil
}

// FIXME: duplicated
func makeJoinableViewName(projectID, datasetID, tableID string) string {
	// datasetID will always be same markedsplassen dataset id
	return fmt.Sprintf("%v_%v", projectID, tableID)
}

func (s *accessService) GrantAccessToDataset(ctx context.Context, user *service.User, input service.GrantAccessData, gcpProjectID string) error {
	const op errs.Op = "accessService.GrantAccessToDataset"

	// FIXME: move this up the call chain
	if input.Expires != nil && input.Expires.Before(time.Now()) {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("expires is in the past"))
	}

	subj := user.Email
	if input.Subject != nil {
		subj = *input.Subject
	}
	ds, err := s.dataProductStorage.GetDataset(ctx, input.DatasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := s.dataProductStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	if err := ensureUserInGroup(user, dp.Owner.Group); err != nil {
		return errs.E(op, err)
	}

	if ds.Pii == "sensitive" && subj == "all-users@nav.no" {
		return errs.E(errs.InvalidRequest, op, fmt.Errorf("datasett som inneholder personopplysninger kan ikke gjøres tilgjengelig for alle interne brukere"))
	}

	bqds, err := s.bigQueryStorage.GetBigqueryDatasource(ctx, ds.ID, false)
	if err != nil {
		return errs.E(op, err)
	}

	subjType := service.SubjectTypeUser
	if input.SubjectType != nil {
		subjType = *input.SubjectType
	}
	subjWithType := subjType + ":" + subj

	owner := subj
	if input.Owner != nil && *input.SubjectType == service.SubjectTypeServiceAccount {
		owner = *input.Owner
	}

	if len(bqds.PseudoColumns) > 0 {
		joinableViews, err := s.joinableViewStorage.GetJoinableViewsForReferenceAndUser(ctx, subj, ds.ID)
		if err != nil {
			return errs.E(op, err)
		}

		for _, jv := range joinableViews {
			joinableViewName := makeJoinableViewName(bqds.ProjectID, bqds.Dataset, bqds.Table)
			if err := s.bigQueryAPI.Grant(ctx, gcpProjectID, jv.Dataset, joinableViewName, subjWithType); err != nil {
				return errs.E(op, err)
			}
		}
	}

	if err := s.bigQueryAPI.Grant(ctx, bqds.ProjectID, bqds.Dataset, bqds.Table, subjWithType); err != nil {
		return errs.E(op, err)
	}

	err = s.accessStorage.GrantAccessToDatasetAndRenew(ctx, input.DatasetID, input.Expires, subjWithType, owner, user.Email)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func ensureOwner(user *service.User, owner string) error {
	const op errs.Op = "ensureOwner"

	if user != nil && (user.GoogleGroups.Contains(owner) || owner == user.Email) {
		return nil
	}

	return errs.E(errs.Unauthorized, op, errs.UserName(user.Email), fmt.Errorf("user is not owner"))
}

func NewAccessService(
	dataCatalogueURL string,
	slackapi service.SlackAPI,
	pollyStorage service.PollyStorage,
	accessStorage service.AccessStorage,
	dataProductStorage service.DataProductsStorage,
	bigQueryStorage service.BigQueryStorage,
	joinableViewStorage service.JoinableViewsStorage,
	bigQueryAPI service.BigQueryAPI,
) *accessService {
	return &accessService{
		dataCatalogueURL:    dataCatalogueURL,
		slackapi:            slackapi,
		pollyStorage:        pollyStorage,
		accessStorage:       accessStorage,
		dataProductStorage:  dataProductStorage,
		bigQueryStorage:     bigQueryStorage,
		joinableViewStorage: joinableViewStorage,
		bigQueryAPI:         bigQueryAPI,
	}
}
