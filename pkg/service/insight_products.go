package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type InsightProductStorage interface {
	GetInsightProductsNumberByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
	GetInsightProductsByTeamID(ctx context.Context, teamIDs []uuid.UUID) ([]*InsightProduct, error)
	GetInsightProductsByGroups(ctx context.Context, groups []string) ([]*InsightProduct, error)
	GetInsightProductWithTeamkatalogen(ctx context.Context, id uuid.UUID) (*InsightProduct, error)
	UpdateInsightProduct(ctx context.Context, id uuid.UUID, in UpdateInsightProductDto) (*InsightProduct, error)
	CreateInsightProduct(ctx context.Context, creator string, in NewInsightProduct) (*InsightProduct, error)
	DeleteInsightProduct(ctx context.Context, id uuid.UUID) error
}

type InsightProductService interface {
	GetInsightProduct(ctx context.Context, id uuid.UUID) (*InsightProduct, error)
	UpdateInsightProduct(ctx context.Context, user *User, id uuid.UUID, input UpdateInsightProductDto) (*InsightProduct, error)
	CreateInsightProduct(ctx context.Context, user *User, input NewInsightProduct) (*InsightProduct, error)
	DeleteInsightProduct(ctx context.Context, user *User, id uuid.UUID) (*InsightProduct, error)
}

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
	TeamID *uuid.UUID `json:"teamID,omitempty"`
	// created is the timestamp for when the insight product was created
	Created time.Time `json:"created"`
	// lastModified is the timestamp for when the insight product was last modified
	LastModified    *time.Time `json:"lastModified,omitempty"`
	TeamName        *string    `json:"teamName"`
	ProductAreaName string     `json:"productAreaName"`
}

type UpdateInsightProductDto struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	TypeArg          string     `json:"type"`
	Link             string     `json:"link"`
	Keywords         []string   `json:"keywords"`
	TeamkatalogenURL *string    `json:"teamkatalogenURL"`
	ProductAreaID    *uuid.UUID `json:"productAreaID"`
	TeamID           *uuid.UUID `json:"teamID"`
	Group            string     `json:"group"`
}

// NewInsightProduct contains the metadata and content of insight products.
type NewInsightProduct struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Type        string   `json:"type"`
	Link        string   `json:"link"`
	Keywords    []string `json:"keywords"`
	// Group is the owner group of the insight product
	Group string `json:"group"`
	// TeamkatalogenURL of the creator
	TeamkatalogenURL *string `json:"teamkatalogenURL,omitempty"`
	// Id of the creator's product area.
	ProductAreaID *uuid.UUID `json:"productAreaID,omitempty"`
	// Id of the creator's team.
	TeamID *uuid.UUID `json:"teamID,omitempty"`
}
