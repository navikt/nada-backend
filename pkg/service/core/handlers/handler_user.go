package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
)

type UserHandler struct {
	service service.UserService
}

func (h *UserHandler) GetUserData(ctx context.Context, _ *http.Request, _ any) (*service.UserInfo, error) {
	user := auth.GetUser(ctx)

	return h.service.GetUserData(ctx, user)
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}
