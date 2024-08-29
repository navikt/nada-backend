package slack

import (
	"fmt"
	"strings"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	slackapi "github.com/slack-go/slack"
)

// FIXME: create an actual slack client

type slackAPI struct {
	webhookURL string
	token      string
	api        *slackapi.Client
}

var _ service.SlackAPI = &slackAPI{}

func (a *slackAPI) SendSlackNotification(channel, message string) error {
	const op = "slackAPI.SendSlackNotification"

	_, _, _, err := a.api.SendMessage(channel, slackapi.MsgOptionText(message, false))
	if err != nil {
		return errs.E(errs.IO, op, err)
	}

	return nil
}

func (a *slackAPI) IsValidSlackChannel(name string) error {
	const op = "slackAPI.IsValidSlackChannel"

	c := ""
	for i := 0; i < 10; i++ {
		chn, nc, e := a.api.GetConversations(&slackapi.GetConversationsParameters{
			Cursor:          c,
			ExcludeArchived: true,
			Types:           []string{"public_channel"},
			Limit:           1000,
		})
		if e != nil {
			return errs.E(errs.IO, op, e)
		}

		for _, cn := range chn {
			if strings.EqualFold(cn.Name, name) {
				return nil
			}
		}

		if nc == "" {
			return errs.E(errs.NotExist, op, fmt.Errorf("channel %s not found", name))
		}

		c = nc
	}

	return errs.E(errs.Internal, op, fmt.Errorf("too many channels to search"))
}

func NewSlackAPI(webhookURL, token string) *slackAPI {
	return &slackAPI{
		webhookURL: webhookURL,
		token:      token,
		api:        slackapi.New(token),
	}
}
