package service

import "context"

type NaisConsoleStorage interface {
	GetAllTeamProjects(ctx context.Context) (map[string]string, error)
	UpdateAllTeamProjects(ctx context.Context, teamProjects map[string]string) error
	GetTeamProject(ctx context.Context, naisTeam string) (string, error)
}

type NaisConsoleAPI interface {
	GetGoogleProjectsForAllTeams(ctx context.Context) (map[string]string, error)
}

type NaisConsoleService interface {
	UpdateAllTeamProjects(ctx context.Context) error
}
