package http

import (
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

var _ service.TeamKatalogenAPI = &teamKatalogenAPI{}

type teamKatalogenAPI struct {
	client *http.Client
	url    string
}

func NewTeamKatalogenAPI() *teamKatalogenAPI {
	return &teamKatalogenAPI{}
}
