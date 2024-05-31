package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
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

type UpdateInsightProductDto struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	TypeArg          string   `json:"type"`
	Link             string   `json:"link"`
	Keywords         []string `json:"keywords"`
	TeamkatalogenURL *string  `json:"teamkatalogenURL"`
	ProductAreaID    *string  `json:"productAreaID"`
	TeamID           *string  `json:"teamID"`
	Group            string   `json:"group"`
}

// NewInsightProduct contains the metadata and content of insight products.
type NewInsightProduct struct {
	// name of the insight product.
	Name string `json:"name"`
	// description of the insight product.
	Description *string `json:"description,omitempty"`
	// type of the insight product.
	Type string `json:"type"`
	// link to the insight product.
	Link string `json:"link"`
	// keywords for the story used as tags.
	Keywords []string `json:"keywords"`
	// group is the owner group of the insight product
	Group string `json:"group"`
	// teamkatalogenURL of the creator
	TeamkatalogenURL *string `json:"teamkatalogenURL,omitempty"`
	// Id of the creator's product area.
	ProductAreaID *string `json:"productAreaID,omitempty"`
	// Id of the creator's team.
	TeamID *string `json:"teamID,omitempty"`
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

func getInsightProduct(ctx context.Context, id string) (*InsightProduct, *APIError) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "Invalid UUID")
	}
	productSQL, err := queries.GetInsightProductWithTeamkatalogen(ctx, productUUID)
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to get insight product")
	}

	return insightProductFromSQL(&productSQL), nil
}

func updateInsightProduct(ctx context.Context, id string, input UpdateInsightProductDto) (*InsightProduct, *APIError) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, NewAPIError(http.StatusBadRequest, err, "Invalid UUID")
	}
	existing, apierr := getInsightProduct(ctx, id)
	if apierr != nil {
		return nil, apierr
	}

	user := auth.GetUser(ctx)
	if !user.GoogleGroups.Contains(existing.Group) {
		return nil, NewAPIError(http.StatusUnauthorized, fmt.Errorf("unauthorized"), "user not in the group of the insight product")
	}

	dbProduct, err := queries.UpdateInsightProduct(ctx, gensql.UpdateInsightProductParams{
		ID:               productUUID,
		Name:             input.Name,
		Description:      ptrToNullString(&input.Description),
		Keywords:         input.Keywords,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		Type:             input.TypeArg,
		Link:             input.Link,
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to update insight product")
	}

	return getInsightProduct(ctx, dbProduct.ID.String())
}

func createInsightProduct(ctx context.Context, input NewInsightProduct) (*InsightProduct, *APIError) {
	creator := auth.GetUser(ctx).Email

	insightProductSQL, err := queries.CreateInsightProduct(ctx, gensql.CreateInsightProductParams{
		Name:             input.Name,
		Creator:          creator,
		Description:      ptrToNullString(input.Description),
		Keywords:         input.Keywords,
		OwnerGroup:       input.Group,
		TeamkatalogenUrl: ptrToNullString(input.TeamkatalogenURL),
		TeamID:           ptrToNullString(input.TeamID),
		Type:             input.Type,
		Link:             input.Link,
	})
	if err != nil {
		return nil, DBErrorToAPIError(err, "Failed to create insight product")
	}

	return getInsightProduct(ctx, insightProductSQL.ID.String())
}
