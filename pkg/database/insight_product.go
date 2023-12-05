package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateInsightProduct(ctx context.Context, creator string,
	newInsightProduct models.NewInsightProduct,
) (*models.InsightProduct, error) {
	insightProductSQL, err := r.querier.CreateInsightProduct(ctx, gensql.CreateInsightProductParams{
		Name:             newInsightProduct.Name,
		Creator:          creator,
		Description:      ptrToNullString(newInsightProduct.Description),
		Keywords:         newInsightProduct.Keywords,
		OwnerGroup:       newInsightProduct.Group,
		TeamkatalogenUrl: ptrToNullString(newInsightProduct.TeamkatalogenURL),
		TeamID:           ptrToNullString(newInsightProduct.TeamID),
		Type:             newInsightProduct.Type,
		Link:             newInsightProduct.Link,
	})
	if err != nil {
		return nil, err
	}

	return InsightProductSQLToGraphql(&insightProductSQL), nil
}

func (r *Repo) GetInsightProduct(ctx context.Context, id uuid.UUID) (*models.InsightProduct, error) {
	productSQL, err := r.querier.GetInsightProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	return InsightProductSQLToGraphql(&productSQL), nil
}

func (r *Repo) GetInsightProducts(ctx context.Context) ([]*models.InsightProduct, error) {
	productSQLs, err := r.querier.GetInsightProducts(ctx)
	if err != nil {
		return nil, err
	}

	productGraphqls := make([]*models.InsightProduct, len(productSQLs))
	for idx, product := range productSQLs {
		productGraphqls[idx] = InsightProductSQLToGraphql(&product)
	}

	return productGraphqls, nil
}

func (r *Repo) UpdateInsightProductMetadata(ctx context.Context, id uuid.UUID, name string,
	description string, keywords []string, teamkatalogenURL *string, productAreaID *string, teamID *string,
	insightProductType string, link string) (
	*models.InsightProduct, error,
) {
	dbProduct, err := r.querier.UpdateInsightProduct(ctx, gensql.UpdateInsightProductParams{
		ID:               id,
		Name:             name,
		Description:      ptrToNullString(&description),
		Keywords:         keywords,
		TeamkatalogenUrl: ptrToNullString(teamkatalogenURL),
		TeamID:           ptrToNullString(teamID),
		Type:             insightProductType,
		Link:             link,
	})
	if err != nil {
		return nil, err
	}

	return InsightProductSQLToGraphql(&dbProduct), nil
}

func (r *Repo) GetInsightProductsByProductArea(ctx context.Context, productAreaID string) ([]*models.InsightProduct, error) {
	dbProducts, err := r.querier.GetInsightProductsByProductArea(ctx, ptrToNullString(&productAreaID))
	if err != nil {
		return nil, err
	}

	graphqlProducts := make([]*models.InsightProduct, len(dbProducts))
	for i, p := range dbProducts {
		graphqlProducts[i] = InsightProductSQLToGraphql(&p)
	}

	return graphqlProducts, nil
}

func (r *Repo) GetInsightProductsByTeam(ctx context.Context, teamID string) ([]*models.InsightProduct, error) {
	dbProducts, err := r.querier.GetInsightProductsByTeam(ctx, ptrToNullString(&teamID))
	if err != nil {
		return nil, err
	}

	graphqlProducts := make([]*models.InsightProduct, len(dbProducts))
	for i, p := range dbProducts {
		graphqlProducts[i] = InsightProductSQLToGraphql(&p)
	}

	return graphqlProducts, nil
}

func (r *Repo) GetInsightProductsByGroups(ctx context.Context, groups []string) ([]*models.InsightProduct, error) {
	dbProducts, err := r.querier.GetInsightProductByGroups(ctx, groups)
	if err != nil {
		return nil, err
	}

	products := make([]*models.InsightProduct, len(dbProducts))
	for idx, p := range dbProducts {
		products[idx] = InsightProductSQLToGraphql(&p)
	}

	return products, nil
}

func (r *Repo) DeleteInsightProduct(ctx context.Context, id uuid.UUID) error {
	return r.querier.DeleteInsightProduct(ctx, id)
}

func InsightProductSQLToGraphql(insightProductSQL *gensql.InsightProduct) *models.InsightProduct {
	return &models.InsightProduct{
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
	}
}
