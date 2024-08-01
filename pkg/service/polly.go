package service

import (
	"context"

	"github.com/google/uuid"
)

type PollyStorage interface {
	CreatePollyDocumentation(ctx context.Context, input PollyInput) (Polly, error)
	GetPollyDocumentation(ctx context.Context, id uuid.UUID) (*Polly, error)
}

type PollyAPI interface {
	SearchPolly(ctx context.Context, q string) ([]*QueryPolly, error)
}

type PollyService interface {
	SearchPolly(ctx context.Context, q string) ([]*QueryPolly, error)
}

type Polly struct {
	ID uuid.UUID `json:"id"`
	QueryPolly
}

type PollyInput struct {
	ID *uuid.UUID `json:"id"`
	QueryPolly
}

type QueryPolly struct {
	ExternalID string `json:"externalID"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}
