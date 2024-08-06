package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/navikt/nada-backend/pkg/errs"

	"github.com/navikt/nada-backend/pkg/service"
)

type SlackHandler struct {
	service service.SlackService
}

type isValidSlackChannelResult struct {
	IsValidSlackChannel bool `json:"isValidSlackChannel"`
}

func (h *SlackHandler) IsValidSlackChannel(_ context.Context, r *http.Request, _ any) (*isValidSlackChannelResult, error) {
	const op errs.Op = "SlackHandler.IsValidSlackChannel"

	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		return nil, errs.E(errs.InvalidRequest, op, errs.Parameter("channel"), fmt.Errorf("channelName is required"))
	}

	err := h.service.IsValidSlackChannel(channelName)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &isValidSlackChannelResult{
		IsValidSlackChannel: true,
	}, nil
}

func NewSlackHandler(service service.SlackService) *SlackHandler {
	return &SlackHandler{
		service: service,
	}
}
