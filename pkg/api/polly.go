package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/polly"
)

type Polly struct {
	ID uuid.UUID `json:"id"`
	polly.QueryPolly
}

type PollyInput struct {
	ID *uuid.UUID `json:"id"`
	polly.QueryPolly
}

func createPollyDocumentation(ctx context.Context, pollyInput PollyInput) (Polly, error) {
	pollyDocumentation, err := queries.CreatePollyDocumentation(ctx, gensql.CreatePollyDocumentationParams{
		ExternalID: pollyInput.ExternalID,
		Name:       pollyInput.Name,
		Url:        pollyInput.URL,
	})
	if err != nil {
		return Polly{}, err
	}

	return Polly{
		ID: pollyDocumentation.ID,
		QueryPolly: polly.QueryPolly{
			ExternalID: pollyDocumentation.ExternalID,
			Name:       pollyDocumentation.Name,
			URL:        pollyDocumentation.Url,
		},
	}, nil
}

func searchPolly(ctx context.Context, q string) ([]*polly.QueryPolly, *APIError) {
	pollyDoc, err := pollyClient.SearchPolly(ctx, q)
	if err != nil {
		return nil, NewAPIError(http.StatusInternalServerError, err, "Failed to search polly")
	}
	return pollyDoc, nil
}
