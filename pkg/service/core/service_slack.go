package core

import (
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type slackService struct {
	slackAPI service.SlackAPI
}

func (s *slackService) IsValidSlackChannel(name string) error {
	const op errs.Op = "slackService.IsValidSlackChannel"

	err := s.slackAPI.IsValidSlackChannel(name)
	if err != nil {
		return errs.E(op, err)
	}

	return nil
}

func NewSlackService(slackAPI service.SlackAPI) service.SlackService {
	return &slackService{
		slackAPI: slackAPI,
	}
}
