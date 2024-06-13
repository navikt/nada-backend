package handlers

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/service/core"
	"net/http"
)

type Endpoints struct {
	GetGCSObject    http.HandlerFunc
	CreateStoryHTTP http.HandlerFunc
	UpdateStoryHTTP http.HandlerFunc
	AppendStoryHTTP http.HandlerFunc
}

func NewEndpoints(h *Handlers) *Endpoints {
	return &Endpoints{
		GetGCSObject:    h.StoryHandler.GetGCSObject,
		CreateStoryHTTP: h.StoryHandler.CreateStoryHTTP,
		UpdateStoryHTTP: h.StoryHandler.UpdateStoryHTTP,
		AppendStoryHTTP: h.StoryHandler.AppendStoryHTTP,
	}
}

type Handlers struct {
	StoryHandler *storyHandler
}

func NewHandlers(s *core.Services, amplitude amplitude.Amplitude) *Handlers {
	return &Handlers{
		StoryHandler: NewStoryHandler(s.StoryService, s.TokenService, amplitude),
	}
}
