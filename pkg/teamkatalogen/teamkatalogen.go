package teamkatalogen

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"time"

	"github.com/sirupsen/logrus"
)

type teamkatalogen struct {
	api     service.TeamKatalogenAPI
	storage service.ProductAreaStorage
	log     *logrus.Logger
}

func New(api service.TeamKatalogenAPI, storage service.ProductAreaStorage, log *logrus.Logger) *teamkatalogen {
	tk := &teamkatalogen{
		api:     api,
		storage: storage,
		log:     log,
	}

	tk.RunSyncer()

	return tk
}

func (t *teamkatalogen) RunSyncer() {
	go func() {
		t.log.Info("Starting Team Katalogen syncer")
		for {
			t.syncTeamkatalogen()
			// FIXME: make this configurable, and use time.Ticker
			time.Sleep(1 * time.Hour)
		}
	}()
}

func (t *teamkatalogen) syncTeamkatalogen() {
	t.log.Info("Syncing Team Katalogen data...")
	pas, err := t.api.GetProductAreas(context.Background())
	if err != nil {
		t.log.WithError(err).Error("Failed to get product areas from Team Katalogen")
		return
	}

	if len(pas) == 0 {
		t.log.Error("No product areas found in Team Katalogen")
		return
	}

	var allTeams []*service.TeamkatalogenTeam

	for _, pa := range pas {
		teams, err := t.api.GetTeamsInProductArea(context.Background(), pa.ID)
		if err != nil {
			t.log.WithError(err).Error("Failed to get teams in product area from Team Katalogen")
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

	err = t.storage.UpsertProductAreaAndTeam(context.Background(), inputProductAreas, inputTeams)
	if err != nil {
		t.log.WithError(err).Error("Failed to upsert product areas and teams")
		return
	}

	t.log.Info("Done syncing Team Katalogen data")
}
