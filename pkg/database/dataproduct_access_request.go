package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateAccessRequestForDataproduct(ctx context.Context, dataproductID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject, owner string) error {
	return r.querier.CreateAccessRequestForDataproduct(ctx, gensql.CreateAccessRequestForDataproductParams{
		DataproductID:        dataproductID,
		Subject:              subject,
		Owner:                owner,
		PollyDocumentationID: pollyDocumentationID,
	})
}

func (r *Repo) ListAccessRequestsForOwner(ctx context.Context, owners []string) ([]*models.AccessRequest, error) {
	accessRequestSQLs, err := r.querier.ListAccessRequestsForOwner(ctx, owners)
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLsToGraphql(ctx, accessRequestSQLs)
}

func (r *Repo) ListAccessRequestsForDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.AccessRequest, error) {
	accessRequestSQLs, err := r.querier.ListAccessRequestsForDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLsToGraphql(ctx, accessRequestSQLs)
}

func (r *Repo) GetAccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	dataproductAccessRequest, err := r.querier.GetAccessRequest(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.accessRequestSQLToGraphql(ctx, dataproductAccessRequest)
}

func (r *Repo) accessRequestSQLsToGraphql(ctx context.Context, accessRequestSQLs []gensql.DataproductAccessRequest) ([]*models.AccessRequest, error) {
	var accessRequests []*models.AccessRequest
	for _, ar := range accessRequestSQLs {
		accessRequestGraphql, err := r.accessRequestSQLToGraphql(ctx, ar)
		if err != nil {
			return nil, err
		}
		accessRequests = append(accessRequests, accessRequestGraphql)
	}
	return accessRequests, nil
}

func (r *Repo) accessRequestSQLToGraphql(ctx context.Context, dataproductAccessRequest gensql.DataproductAccessRequest) (*models.AccessRequest, error) {
	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject)
	}
	subject := splits[1]
	subjectType := models.StringToSubjectType(splits[0])

	polly, err := r.pollySQLToGraphql(ctx, dataproductAccessRequest.PollyDocumentationID)
	if err != nil {
		return nil, err
	}

	return &models.AccessRequest{
		DataproductID: dataproductAccessRequest.DataproductID,
		Subject:       &subject,
		SubjectType:   &subjectType,
		Polly:         polly,
	}, nil
}

func (r *Repo) pollySQLToGraphql(ctx context.Context, id uuid.NullUUID) (*models.DatabasePolly, error) {
	var polly models.DatabasePolly
	if id.Valid {
		pollyDoc, err := r.querier.GetPollyDocumentation(ctx, id.UUID)
		if err != nil {
			return nil, err
		}

		polly.ID = pollyDoc.ID
		polly.URL = pollyDoc.Url
		polly.ExternalID = pollyDoc.ExternalID
		polly.Name = pollyDoc.Name
	}
	return &polly, nil
}
