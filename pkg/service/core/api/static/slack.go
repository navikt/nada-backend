package static

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type slackAPI struct {
	log zerolog.Logger
}

func (s *slackAPI) InformNewAccessRequest(_ context.Context, subject string, datasetID uuid.UUID) error {
	s.log.Info().Msgf("Informing new access request for %s to dataset %s", subject, datasetID.String())

	return nil
}

func (s *slackAPI) IsValidSlackChannel(channel string) error {
	s.log.Info().Msgf("Validating slack channel %s", channel)

	return nil
}

func NewSlackAPI(log zerolog.Logger) *slackAPI {
	return &slackAPI{
		log: log,
	}
}
