package teamkatalogen

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"
)

type Syncer struct {
	api     service.TeamKatalogenAPI
	storage service.ProductAreaStorage
	log     zerolog.Logger
}

func New(api service.TeamKatalogenAPI, storage service.ProductAreaStorage, log zerolog.Logger) *Syncer {
	tk := &Syncer{
		api:     api,
		storage: storage,
		log:     log,
	}

	return tk
}

func (s *Syncer) Run(ctx context.Context, frequency time.Duration) {
	s.log.Info().Msg("Starting Team Katalogen syncer")

	ticker := time.NewTicker(frequency)

	s.RunOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunOnce()
		}
	}
}

func (s *Syncer) RunOnce() {
	s.log.Info().Msg("Syncing Team Katalogen data...")

	pas, err := s.api.GetProductAreas(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("getting product areas from Team Katalogen")
		return
	}

	if len(pas) == 0 {
		s.log.Info().Msg("No product areas found in Team Katalogen")
		return
	}

	var allTeams []*service.TeamkatalogenTeam

	for _, pa := range pas {
		teams, err := s.api.GetTeamsInProductArea(context.Background(), pa.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("getting teams in product area from Team Katalogen")
			return
		}
		allTeams = append(allTeams, teams...)
	}

	inputTeams := make([]*service.UpsertTeamRequest, len(allTeams))
	for i, team := range allTeams {
		inputTeams[i] = &service.UpsertTeamRequest{
			ID:            team.ID,
			ProductAreaID: team.ProductAreaID,
			Name:          team.Name,
		}
	}

	inputProductAreas := make([]*service.UpsertProductAreaRequest, len(pas))
	for i, pa := range pas {
		inputProductAreas[i] = &service.UpsertProductAreaRequest{
			ID:   pa.ID,
			Name: pa.Name,
		}
	}

	err = s.storage.UpsertProductAreaAndTeam(context.Background(), inputProductAreas, inputTeams)
	if err != nil {
		s.log.Error().Err(err).Msg("upsert product areas and teams")
		return
	}

	s.log.Info().Msg("done syncing Team Katalogen data")
}
