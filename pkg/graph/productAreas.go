package graph

import "context"

func (r *productAreaResolver) teamsInPA(ctx context.Context, paID string) ([]string, error) {
	teamsInPA, err := r.teamkatalogen.GetTeamsInProductArea(ctx, paID)
	if err != nil {
		return nil, err
	}

	teamIDsInPA := []string{}
	for _, team := range teamsInPA {
		teamIDsInPA = append(teamIDsInPA, team.ID)
	}

	return teamIDsInPA, nil
}
