package slack

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	slackapi "github.com/slack-go/slack"
	"net/url"
	"strings"
)

// FIXME: create an actual slack client

type slackAPI struct {
	webhookURL          string
	dataCatalogueURL    string
	token               string
	api                 *slackapi.Client
	dataProductsStorage service.DataProductsStorage
}

var _ service.SlackAPI = &slackAPI{}

func (a *slackAPI) InformNewAccessRequest(ctx context.Context, subject string, datasetID uuid.UUID) error {
	const op = "slackAPI.InformNewAccessRequest"

	ds, err := a.dataProductsStorage.GetDataset(ctx, datasetID)
	if err != nil {
		return errs.E(op, err)
	}

	dp, err := a.dataProductsStorage.GetDataproduct(ctx, ds.DataproductID)
	if err != nil {
		return errs.E(op, err)
	}

	if dp.Owner.TeamContact == nil || *dp.Owner.TeamContact == "" {
		// Access request created but skip Slack message because team contact is empty
		return nil
	}

	link := fmt.Sprintf(
		"\nLink: %s/dataproduct/%s/%s/%s",
		a.dataCatalogueURL,
		dp.ID.String(),
		url.QueryEscape(dp.Name),
		ds.ID.String(),
	)

	dsp := fmt.Sprintf(
		"\nDatasett: %s\nDataprodukt: %s",
		ds.Name,
		dp.Name,
	)

	message := fmt.Sprintf(
		"%s har sendt en søknad om tilgang for: %s%s",
		subject,
		dsp,
		link,
	)

	_, _, _, err = a.api.SendMessage(*dp.Owner.TeamContact, slackapi.MsgOptionText(message, false))
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

func NewSlackAPI(webhookURL, dataCatalogueURL, token string) *slackAPI {
	return &slackAPI{
		webhookURL:       webhookURL,
		dataCatalogueURL: dataCatalogueURL,
		token:            token,
		api:              slackapi.New(token),
	}
}
