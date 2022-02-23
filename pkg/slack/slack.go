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

func (s SlackClient) NewDataproduct(dp *models.Dataproduct) error {
	var desc string
	if dp.Description != nil {
		desc = *dp.Description
	}
	var owner string
	var link string
	if dp.Owner != nil {
		owner = dp.Owner.Group
		if dp.Owner.TeamkatalogenURL != nil {
			link = " (" + *dp.Owner.TeamkatalogenURL + ")"
		}

	} else {
		owner = "Noen"
		link = ""
	}
	message :=
		owner + link + " har lagd et dataprodukt \nNavn: " + dp.Name + ", beskrivelse: " + desc + "\nLink: " + s.datakatalogurl + "/dataproduct/" + dp.ID.String()

	err := slack.PostWebhook(s.webhookurl, &slack.WebhookMessage{
		Username: "NadaBot",
		Text:     message,
	})
	if err != nil {
		return fmt.Errorf("could not post message to slack %e", err)
	}
	return nil
}
