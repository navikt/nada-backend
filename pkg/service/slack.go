package service

import (
	"context"
)

type SlackAPI interface {
	InformNewAccessRequest(ctx context.Context, subject, datasetID string) error
	IsValidSlackChannel(name string) error
}
