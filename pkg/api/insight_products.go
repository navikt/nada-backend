package api

import (
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

// InsightProduct contains the metadata of insight product.
type InsightProduct struct {
	// id of the insight product.
	ID uuid.UUID `json:"id"`
	// name of the insight product.
	Name string `json:"name"`
	// creator of the insight product.
	Creator string `json:"creator"`
	// description of the insight product.
	Description string `json:"description"`
	// type of the insight product.
	Type string `json:"type"`
	// link to the insight product.
	Link string `json:"link"`
	// keywords for the insight product used as tags.
	Keywords []string `json:"keywords"`
	// group is the owner group of the insight product
	Group string `json:"group"`
	// teamkatalogenURL of the creator
	TeamkatalogenURL *string `json:"teamkatalogenURL,omitempty"`
	// Id of the creator's team.
	TeamID *string `json:"teamID,omitempty"`
	// created is the timestamp for when the insight product was created
	Created time.Time `json:"created"`
	// lastModified is the timestamp for when the insight product was last modified
	LastModified    *time.Time `json:"lastModified,omitempty"`
	TeamName        *string    `json:"teamName"`
	ProductAreaName string     `json:"productAreaName"`
}

func insightProductFromSQL(insightProductSQL *gensql.InsightProductWithTeamkatalogenView) *InsightProduct {
	return &InsightProduct{
		ID:               insightProductSQL.ID,
		Name:             insightProductSQL.Name,
		Creator:          insightProductSQL.Creator,
		Created:          insightProductSQL.Created,
		Description:      insightProductSQL.Description.String,
		Type:             insightProductSQL.Type,
		Keywords:         insightProductSQL.Keywords,
		TeamkatalogenURL: nullStringToPtr(insightProductSQL.TeamkatalogenUrl),
		TeamID:           nullStringToPtr(insightProductSQL.TeamID),
		Group:            insightProductSQL.Group,
		Link:             insightProductSQL.Link,
		TeamName:         nullStringToPtr(insightProductSQL.TeamName),
		ProductAreaName:  nullStringToString(insightProductSQL.PaName),
	}
}
