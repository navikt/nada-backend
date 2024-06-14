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

func (h *slackHandler) IsValidSlackChannel(_ context.Context, r *http.Request, _ any) (*Empty, error) {
	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		return nil, fmt.Errorf("channelName is required")
	}

	return &Empty{}, h.service.IsValidSlackChannel(channelName)
}

func NewSlackHandler(service service.SlackService) *slackHandler {
	return &slackHandler{
		service: service,
	}
}
