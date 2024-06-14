package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type userHandler struct {
	service service.UserService
}

func (h *userHandler) GetUserData(ctx context.Context, _ *http.Request, _ any) (*service.UserInfo, error) {
	// FIXME: read out the user here
	return h.service.GetUserData(ctx)
}

func NewUserHandler(service service.UserService) *userHandler {
	return &userHandler{service: service}
}
