package handlers

import (
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type slackHandler struct {
	service service.SlackService
}

type isValidSlackChannelResult struct {
	IsValidSlackChannel bool `json:"isValidSlackChannel"`
}

func (h *slackHandler) IsValidSlackChannel(_ context.Context, r *http.Request, _ any) (*isValidSlackChannelResult, error) {
	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		return nil, fmt.Errorf("channelName is required")
	}

	err := h.service.IsValidSlackChannel(channelName)
	if err != nil {
		return nil, err
	}

	return &isValidSlackChannelResult{
		IsValidSlackChannel: true,
	}, nil
}

func NewSlackHandler(service service.SlackService) *slackHandler {
	return &slackHandler{
		service: service,
	}
}
