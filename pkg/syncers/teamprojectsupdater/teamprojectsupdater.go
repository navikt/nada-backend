package teamprojectsupdater

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/navikt/nada-backend/pkg/leaderelection"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
)

var ErrNotLeader = fmt.Errorf("not leader")

type TeamProjectsUpdater struct {
	service service.NaisConsoleService
	log     zerolog.Logger
}

func New(service service.NaisConsoleService, log zerolog.Logger) *TeamProjectsUpdater {
	return &TeamProjectsUpdater{
		service: service,
		log:     log,
	}
}

func (t *TeamProjectsUpdater) Run(ctx context.Context, startupDelay, frequency time.Duration) {
	t.log.Info().Dur("update_frequency", frequency).Msg("starting team projects updater")

	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	time.Sleep(startupDelay)

	t.log.Info().Msg("running initial update")

	err := t.RunOnce(ctx)
	if err != nil {
		if errors.Is(err, ErrNotLeader) {
			t.log.Info().Msg("not leader, skipping update")
		} else {
			t.log.Error().Err(err).Msg("updating team projects")
		}
	}

	t.log.Info().Msg("initial update done")

	for {
		select {
		case <-ticker.C:
			t.log.Info().Msg("updating team projects")

			err := t.RunOnce(ctx)
			if err != nil {
				if errors.Is(err, ErrNotLeader) {
					t.log.Info().Msg("not leader, skipping update")
					continue
				}

				t.log.Error().Err(err).Msg("updating team projects")
			}

			t.log.Info().Msg("team projects updated")
		case <-ctx.Done():
			return
		}
	}
}

func (t *TeamProjectsUpdater) RunOnce(ctx context.Context) error {
	isLeader, err := leaderelection.IsLeader()
	if err != nil {
		return fmt.Errorf("checking leader status: %w", err)
	}

	if !isLeader {
		return ErrNotLeader
	}

	err = t.service.UpdateAllTeamProjects(ctx)
	if err != nil {
		return fmt.Errorf("invoking service: %w", err)
	}

	return nil
}
