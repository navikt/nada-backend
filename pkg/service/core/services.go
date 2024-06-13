package core

import "github.com/navikt/nada-backend/pkg/service"

type Services struct {
	StoryService service.StoryService
	UserService  service.UserService
	TokenService service.TokenService
}

func NewServices(
	storyService service.StoryService,
	userService service.UserService,
	tokenService service.TokenService,
) *Services {
	return &Services{
		StoryService: storyService,
		UserService:  userService,
		TokenService: tokenService,
	}
}
