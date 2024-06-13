package handlers

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/service/core"
	"net/http"
)

// Inspired by: https://www.willem.dev/articles/generic-http-handlers/

type Endpoints struct {
	GetGCSObject      http.HandlerFunc
	CreateStoryHTTP   http.HandlerFunc
	UpdateStoryHTTP   http.HandlerFunc
	AppendStoryHTTP   http.HandlerFunc
	GetAllTeamTokens  http.HandlerFunc
	GetDataProduct    http.HandlerFunc
	CreateDataProduct http.HandlerFunc
	DeleteDataProduct http.HandlerFunc
	UpdateDataProduct http.HandlerFunc
}

func NewEndpoints(h *Handlers) *Endpoints {
	return &Endpoints{
		// Story endpoints
		GetGCSObject:    h.StoryHandler.GetGCSObject,
		CreateStoryHTTP: h.StoryHandler.CreateStoryHTTP,
		UpdateStoryHTTP: h.StoryHandler.UpdateStoryHTTP,
		AppendStoryHTTP: h.StoryHandler.AppendStoryHTTP,

		// Token endpoints
		GetAllTeamTokens: h.TokenHandler.GetAllTeamTokens,

		// Data product endpoints
		GetDataProduct:    HandlerFor(h.DataProductsHandler.GetDataProduct).ResponseToJSON().Build(),
		CreateDataProduct: HandlerFor(h.DataProductsHandler.CreateDataProduct).RequestFromJSON().ResponseToJSON().Build(),
		DeleteDataProduct: HandlerFor(h.DataProductsHandler.DeleteDataProduct).ResponseToJSON().Build(),
		UpdateDataProduct: HandlerFor(h.DataProductsHandler.UpdateDataProduct).RequestFromJSON().ResponseToJSON().Build(),
	}
}

type Handlers struct {
	StoryHandler        *storyHandler
	TokenHandler        *tokenHandler
	DataProductsHandler *dataProductsHandler
}

func NewHandlers(s *core.Services, amplitude amplitude.Amplitude) *Handlers {
	return &Handlers{
		StoryHandler: NewStoryHandler(s.StoryService, s.TokenService, amplitude),
	}
}
