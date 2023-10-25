package slack

import (
	"fmt"
	"strings"

	"github.com/navikt/nada-backend/pkg/graph/models"
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

func (s SlackClient) NewAccessRequest(contact string, dp *models.Dataproduct, ds *models.Dataset, ar *models.AccessRequest) error {
	link := "\nLink: " + s.datakatalogurl + "/dataproduct/" + dp.ID.String() + "/" + strings.ReplaceAll(dp.Name, " ", "%20") + "/" + ds.ID.String()
	dsp := "\nDatasett: " + ds.Name + " " + "\nDataprodukt: " + dp.Name
	message := ar.Subject + " har sendt en søknad om tilgang for: " + dsp + link
	_, _, _, e := s.api.SendMessage(contact, slack.MsgOptionText(message, false))
	return e
}

func (s SlackClient) IsValidSlackChannel(name string) (bool, error) {
	chn, e := s.GetPublicChannel(name)
	return chn != nil, e
}

func (s SlackClient) GetPublicChannel(name string) (*slack.Channel, error) {

	c := ""
	for i := 0; i < 10; i++ {
		chn, nc, e := s.api.GetConversations(&slack.GetConversationsParameters{
			Cursor:          c,
			ExcludeArchived: true,
			Types:           []string{"public_channel"},
			Limit:           500,
		})
		if e != nil {
			return nil, e
		}

		for _, cn := range chn {
			if strings.ToLower(cn.Name) == strings.ToLower(name) {
				return &cn, nil
			}
		}

		if nc == "" {
			return nil, nil
		}
		c = nc
	}
	return nil, fmt.Errorf("Too many channels in workspace")
}
