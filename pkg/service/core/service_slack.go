package core

import "github.com/navikt/nada-backend/pkg/service"

type slackService struct {
	slackAPI service.SlackAPI
}

func (s *slackService) IsValidSlackChannel(name string) error {
	return s.slackAPI.IsValidSlackChannel(name)
}

func NewSlackService(slackAPI service.SlackAPI) service.SlackService {
	return &slackService{
		slackAPI: slackAPI,
	}
}
