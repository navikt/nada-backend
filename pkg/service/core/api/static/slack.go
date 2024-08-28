package static

import (
	"github.com/rs/zerolog"
)

type slackAPI struct {
	log zerolog.Logger
}

func (s *slackAPI) SendSlackNotification(channel, message string) error {
	s.log.Info().Msgf("Sending slack notification to channel %v: message: %v", channel, message)

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
