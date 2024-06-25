package service

import (
	"context"
	"github.com/google/uuid"
)

type SlackAPI interface {
	InformNewAccessRequest(ctx context.Context, subject string, datasetID uuid.UUID) error
	IsValidSlackChannel(name string) error
}

type SlackService interface {
	IsValidSlackChannel(name string) error
}
