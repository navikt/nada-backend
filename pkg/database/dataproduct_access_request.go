package database

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"strings"
)

func (r *Repo) CreateAccessRequestForDataproduct(ctx context.Context, dataproductID uuid.UUID, pollyDocumentationID uuid.NullUUID, subject string) error {
	return r.querier.CreateAccessRequestForDataproduct(ctx, gensql.CreateAccessRequestForDataproductParams{
		DataproductID:        dataproductID,
		Subject:              subject,
		PollyDocumentationID: pollyDocumentationID,
	})
}

func (r *Repo) ListAccessRequestsForDataproduct() {

}

func (r *Repo) ListAccessRequestsForUser() {

}

func (r *Repo) GetAccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	dataproductAccessRequest, err := r.querier.GetAccessRequest(ctx, id)
	if err != nil {
		return nil, err
	}

	splits := strings.Split(dataproductAccessRequest.Subject, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("%v is not a valid subject (can't split on :)", dataproductAccessRequest.Subject)
	}
	subject := splits[0]
	subjectType := models.StringToSubjectType(splits[1])

	var polly models.DatabasePolly
	if dataproductAccessRequest.PollyDocumentationID.Valid {
		pollyDoc, err := r.querier.GetPollyDocumentation(ctx, dataproductAccessRequest.PollyDocumentationID.UUID)
		if err != nil {
			return nil, err
		}

		polly.ID = pollyDoc.ID
		polly.URL = pollyDoc.Url
		polly.ExternalID = pollyDoc.ExternalID
		polly.Name = pollyDoc.Name
	}

	return &models.AccessRequest{
		DataproductID: dataproductAccessRequest.DataproductID,
		Subject:       &subject,
		SubjectType:   &subjectType,
		Polly:         &polly,
	}, nil
}
