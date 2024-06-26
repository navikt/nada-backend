package slack

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type SlackClient struct {
	log            *logrus.Logger
	webhookurl     string
	datakatalogurl string
	token          string
	api            *slack.Client
}

func NewSlackClient(log *logrus.Logger, webhookurl string, datakatalogurl string, token string) *SlackClient {
	return &SlackClient{
		log:            log,
		webhookurl:     webhookurl,
		datakatalogurl: datakatalogurl,
		token:          token,
		api:            slack.New(token),
	}
}

func (s SlackClient) InformNewAccessRequest(contact string, dpID, dpName, dsID, dsName, subject string) error {
	link := "\nLink: " + s.datakatalogurl + "/dataproduct/" + dpID + "/" + strings.ReplaceAll(dpName, " ", "%20") + "/" + dsID
	dsp := "\nDatasett: " + dsName + " " + "\nDataprodukt: " + dpName
	message := subject + " har sendt en søknad om tilgang for: " + dsp + link
	_, _, _, e := s.api.SendMessage(contact, slack.MsgOptionText(message, false))
	return e
}

func (s SlackClient) IsValidSlackChannel(name string) (bool, error) {
	chn, e := s.GetPublicChannel(name)
	return chn != nil, e
}

func (s SlackClient) GetPublicChannel(name string) (*slack.Channel, error) {

	//TODO: the implementation is fragile, but donno how to deal with it
	c := ""
	for i := 0; i < 10; i++ {
		chn, nc, e := s.api.GetConversations(&slack.GetConversationsParameters{
			Cursor:          c,
			ExcludeArchived: true,
			Types:           []string{"public_channel"},
			Limit:           1000,
		})
		if e != nil {
			return nil, e
		}

		for _, cn := range chn {
			if strings.EqualFold(cn.Name, name) {
				return &cn, nil
			}
		}

		if nc == "" {
			return nil, nil
		}
		c = nc
	}
	return nil, fmt.Errorf("too many channels in workspace")
}
