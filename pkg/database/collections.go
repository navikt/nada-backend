package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetCollections(ctx context.Context, limit, offset int) ([]*models.Collection, error) {
	collections := []*models.Collection{}

	res, err := r.querier.GetCollections(ctx, gensql.GetCollectionsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting collections from database: %w", err)
	}

	for _, entry := range res {
		col := collectionFromSQL(entry)
		collections = append(collections, col)
	}

	return collections, nil
}

func (r *Repo) GetCollectionsForElement(ctx context.Context, limit, offset int) ([]*models.Collection, error) {
	collections := []*models.Collection{}

	res, err := r.querier.GetCollectionsForElement(ctx, gensql.GetCollectionsForElementParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting collections for element from database: %w", err)
	}

	for _, entry := range res {
		col := collectionFromSQL(entry)
		collections = append(collections, col)
	}

	return collections, nil
}

func (r *Repo) GetCollection(ctx context.Context, id uuid.UUID) (*models.Collection, error) {
	res, err := r.querier.GetCollection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	col := collectionFromSQL(res)

	return col, nil
}

func (r *Repo) CreateCollection(ctx context.Context, col models.NewCollection) (*models.Collection, error) {
	var keywords []string
	if col.Keywords != nil {
		keywords = []string{}
	}
	res, err := r.querier.CreateCollection(ctx, gensql.CreateCollectionParams{
		Name:        col.Name,
		Description: ptrToNullString(col.Description),
		Slug:        slugify(col.Slug, col.Name),
		OwnerGroup:  col.Group,
		Keywords:    keywords,
	})
	if err != nil {
		return nil, err
	}

	return collectionFromSQL(res), nil
}

func (r *Repo) UpdateCollection(ctx context.Context, id uuid.UUID, new models.UpdateCollection) (*models.Collection, error) {
	var keywords []string
	if new.Keywords != nil {
		keywords = new.Keywords
	}

	res, err := r.querier.UpdateCollection(ctx, gensql.UpdateCollectionParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		ID:          id,
		Keywords:    keywords,
		Slug:        slugify(new.Slug, new.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("updating collection in database: %w", err)
	}

	return collectionFromSQL(res), nil
}

func (r *Repo) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	if err := r.querier.DeleteCollection(ctx, id); err != nil {
		return fmt.Errorf("deleting dataproduct_collection from database: %w", err)
	}

	return nil
}

func (r *Repo) GetCollectionElements(ctx context.Context, id uuid.UUID) ([]models.CollectionElement, error) {
	colElems, err := r.querier.GetCollectionElements(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting collection elements: %w", err)
	}

	ret := []models.CollectionElement{}

	for _, elem := range colElems {
		ret = append(ret, dataproductFromSQL(elem))
	}
	return ret, nil
}

func (r *Repo) AddToCollection(ctx context.Context, id uuid.UUID, elementID uuid.UUID, elementType string) error {
	return r.querier.CreateCollectionElement(ctx, gensql.CreateCollectionElementParams{
		CollectionID: id,
		ElementID:    elementID,
		ElementType:  elementType,
	})
}

func (r *Repo) RemoveFromCollection(ctx context.Context, id uuid.UUID, elementID uuid.UUID, elementType string) error {
	return r.querier.DeleteCollectionElement(ctx, gensql.DeleteCollectionElementParams{
		CollectionID: id,
		ElementID:    elementID,
		ElementType:  elementType,
	})
}

func collectionFromSQL(col gensql.Collection) *models.Collection {
	return &models.Collection{
		ID:           col.ID,
		Name:         col.Name,
		Created:      col.Created,
		LastModified: col.LastModified,
		Description:  nullStringToPtr(col.Description),
		Keywords:     col.Keywords,
		Owner: models.Owner{
			Group: col.Group,
		},
		Slug: col.Slug,
	}
}
