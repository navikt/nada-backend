package slack

import (
	"fmt"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type SlackClient struct {
	log            *logrus.Logger
	webhookurl     string
	datakatalogurl string
}

func NewSlackClient(log *logrus.Logger, webhookurl string, datakatalogurl string) *SlackClient {
	return &SlackClient{
		log:            log,
		webhookurl:     webhookurl,
		datakatalogurl: datakatalogurl,
	}
}
func (s SlackClient) NewDataProdukt(dp *models.Dataproduct) error {
	message :=
		"Noen har lagd et dataprodukt \nNavn: " + dp.Name + ", beskrivelse: " + *dp.Description + "\nLink: " + s.datakatalogurl + "/dataproducts/" + dp.ID.String()

	err := slack.PostWebhook(s.webhookurl, &slack.WebhookMessage{
		Username: "NadaBot",
		Text:     message,
	})
	if err != nil {
		return fmt.Errorf("could not post message to slack %e", err)
	}
	return nil
}
