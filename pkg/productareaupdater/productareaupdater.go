package productareaupdater

import (
	"context"
	"time"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/sirupsen/logrus"
)

type ProductAreaUpdater struct {
	repo              *database.Repo
	teamcatalogClient graph.Teamkatalogen
	log               *logrus.Entry
}

func New(repo *database.Repo, teamcatalogClient graph.Teamkatalogen, log *logrus.Entry) *ProductAreaUpdater {
	return &ProductAreaUpdater{
		repo:              repo,
		teamcatalogClient: teamcatalogClient,
		log:               log,
	}
}

func (p *ProductAreaUpdater) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		p.run(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *ProductAreaUpdater) run(ctx context.Context) {
	teamPAMappings, err := p.repo.GetTeamsAndProductAreaIDs(ctx)
	if err != nil {
		p.log.WithError(err).Error("fetching team and product area mappings")
		return
	}

	for _, teamPAMapping := range teamPAMappings {
		team, err := p.teamcatalogClient.GetTeam(ctx, teamPAMapping.TeamID)
		if err != nil {
			p.log.WithError(err).Errorf("reading team %v from teamkatalogen", teamPAMapping.TeamID)
			continue
		}
		if !teamPAMapping.ProductAreaID.Valid || teamPAMapping.ProductAreaID.String != team.ProductAreaID {
			if err := p.repo.UpdateProductAreaForTeam(ctx, team.ID, team.ProductAreaID); err != nil {
				p.log.WithError(err).Errorf("updating product area id from %v to %v for team %v", teamPAMapping.ProductAreaID, team.ProductAreaID, team.ID)
			}
		}
	}
}
