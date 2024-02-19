package teamkatalogen

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

func (t *teamkatalogen) RunSyncer() {
	go func() {
		t.log.Info("Starting Team Katalogen syncer")
		for {
			// call syncTeamkatalogen() every 1 hour
			t.syncTeamkatalogen()
			time.Sleep(1 * time.Hour)
		}
	}()
}

func (t *teamkatalogen) syncTeamkatalogen() {
	t.log.Info("Syncing Team Katalogen data...")
	pas, err := t.restGetProductAreas(context.Background())
	if err != nil {
		t.log.WithError(err).Error("Failed to get product areas from Team Katalogen")
		return
	}

	if len(pas) == 0 {
		t.log.Error("No product areas found in Team Katalogen")
		return
	}

	var allTeams []*Team

	for _, pa := range pas {
		teams, err := t.GetTeamsInProductArea(context.Background(), pa.ID)
		if err != nil {
			t.log.WithError(err).Error("Failed to get teams in product area from Team Katalogen")
			return
		}
		allTeams = append(allTeams, teams...)
	}

	tx, err := t.db.Begin()
	if err != nil {
		t.log.WithError(err).Error("Failed to start transaction")
		return
	}

	for _, pa := range pas {
		paUUID, err := uuid.Parse(pa.ID)
		if err != nil {
			t.log.WithError(err).Error("Failed to parse product area id: " + pa.ID)
			continue
		}

		err = t.querier.UpsertProductArea(context.Background(), gensql.UpsertProductAreaParams{
			ID:   paUUID,
			Name: pa.Name,
		})
		if err != nil {
			t.log.WithError(err).Error("Failed to upsert product area")
			return
		}
	}

	for _, team := range allTeams {
		teamUUID, err := uuid.Parse(team.ID)
		if err != nil {
			t.log.WithError(err).Error("Failed to parse team id: " + team.ID)
			continue
		}

		err = t.querier.UpsertTeam(context.Background(), gensql.UpsertTeamParams{
			ID:            teamUUID,
			ProductAreaID: uuid.NullUUID{UUID: uuid.MustParse(team.ProductAreaID), Valid: true},
			Name:          team.Name,
		})
		if err != nil {
			t.log.WithError(err).Error("Failed to upsert team")
			continue
		}
	}

	defer tx.Rollback()

	tx.Commit()
	t.log.Info("Done syncing Team Katalogen data")
}
