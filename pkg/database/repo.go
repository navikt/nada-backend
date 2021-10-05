package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"

	// Pin version of sqlc and goose for cli
	_ "github.com/kyleconroy/sqlc"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Repo struct {
	querier gensql.Querier
}

func New(dbConnDSN string) (*Repo, error) {
	db, err := sql.Open("postgres", dbConnDSN)
	if err != nil {
		return nil, fmt.Errorf("open sql connection: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("goose up: %w", err)
	}

	return &Repo{
		querier: gensql.New(db),
	}, nil
}

func slugify(maybeslug *string, fallback string) string {
	if maybeslug != nil {
		return *maybeslug
	}
	// TODO(thokra): Smartify this?
	return url.PathEscape(fallback)
}

func (r *Repo) CreateDataproduct(ctx context.Context, dp openapi.NewDataproduct) (*openapi.Dataproduct, error) {
	var keywords []string
	if dp.Keywords != nil {
		keywords = *dp.Keywords
	}
	res, err := r.querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:        dp.Name,
		Description: ptrToNullString(dp.Description),
		Slug:        slugify(dp.Slug, dp.Name),
		Repo:        ptrToNullString(dp.Repo),
		Team:        dp.Owner.Team,
		Keywords:    keywords,
	})
	if err != nil {
		return nil, err
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) GetDataproducts(ctx context.Context, limit, offset int) ([]*openapi.Dataproduct, error) {
	dataproducts := []*openapi.Dataproduct{}

	res, err := r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts from database: %w", err)
	}

	for _, entry := range res {
		dataproduct := dataproductFromSQL(entry)
		if err := r.enrichDataproduct(ctx, entry.ID, dataproduct); err != nil {
			return nil, err
		}
		dataproducts = append(dataproducts, dataproduct)
	}

	return dataproducts, nil
}

func (r *Repo) enrichDataproduct(ctx context.Context, id uuid.UUID, dp *openapi.Dataproduct) error {
	datasets, err := r.querier.GetDatasetsForDataproduct(ctx, id)
	if err != nil {
		return fmt.Errorf("getting datasets for enriching dataproduct: %w", err)
	}

	for _, v := range datasets {
		dp.Datasets = append(dp.Datasets, openapi.DatasetSummary{
			Id:   v.ID.String(),
			Name: v.Name,
			Type: openapi.DatasetType(v.Type),
		})
	}
	return nil
}

func (r *Repo) GetDataproduct(ctx context.Context, id string) (*openapi.Dataproduct, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}
	res, err := r.querier.GetDataproduct(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	dp := dataproductFromSQL(res)
	if err := r.enrichDataproduct(ctx, uid, dp); err != nil {
		return nil, err
	}

	return dp, nil
}

func (r *Repo) DeleteDataproduct(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parsing uuid: %w", err)
	}

	if err := r.querier.DeleteDataproduct(ctx, uid); err != nil {
		return fmt.Errorf("deleting dataproduct from database: %w", err)
	}

	return nil
}

func (r *Repo) UpdateDataproduct(ctx context.Context, id string, new openapi.NewDataproduct) (*openapi.Dataproduct, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	var keywords []string
	if new.Keywords != nil {
		keywords = *new.Keywords
	}

	res, err := r.querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		Slug:        slugify(new.Slug, new.Name),
		Repo:        ptrToNullString(new.Repo),
		Team:        new.Owner.Team,
		Keywords:    keywords,
		ID:          uid,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) CreateDataset(ctx context.Context, ds openapi.NewDataset) (*openapi.Dataset, error) {
	uid, err := uuid.Parse(ds.DataproductId)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	res, err := r.querier.CreateDataset(ctx, gensql.CreateDatasetParams{
		Name:          ds.Name,
		DataproductID: uid,
		Description:   ptrToNullString(ds.Description),
		Pii:           ds.Pii,
		ProjectID:     ds.Bigquery.ProjectId,
		Dataset:       ds.Bigquery.Dataset,
		TableName:     ds.Bigquery.Table,
		Type:          "bigquery",
	})
	if err != nil {
		return nil, err
	}

	return datasetFromSQL(res), nil
}

func (r *Repo) GetDataset(ctx context.Context, id string) (*openapi.Dataset, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}
	res, err := r.querier.GetDataset(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("getting dataset from database: %w", err)
	}

	return datasetFromSQL(res), nil
}

func (r *Repo) UpdateDataset(ctx context.Context, id string, new openapi.NewDataset) (*openapi.Dataset, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	dataproductUid, err := uuid.Parse(new.DataproductId)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	res, err := r.querier.UpdateDataset(ctx, gensql.UpdateDatasetParams{
		Name:          new.Name,
		Description:   ptrToNullString(new.Description),
		ID:            uid,
		DataproductID: dataproductUid,
		Pii:           new.Pii,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return datasetFromSQL(res), nil
}

func (r *Repo) DeleteDataset(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parsing uuid: %w", err)
	}

	if err := r.querier.DeleteDataset(ctx, uid); err != nil {
		return fmt.Errorf("deleting dataset from database: %w", err)
	}

	return nil
}

func (r *Repo) Search(ctx context.Context, query string, limit, offset int) ([]*openapi.SearchResultEntry, error) {
	results := []*openapi.SearchResultEntry{}
	makeExcerpt := func(s sql.NullString) string {
		if s.Valid {
			return s.String
		}
		return "No description"
	}

	// If query is empty, the search result is empty. So we do a regular SELECT * instead
	var (
		dataproducts []gensql.Dataproduct
		datasets     []gensql.Dataset
		err          error
	)
	if query == "" {
		dataproducts, err = r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	} else {
		dataproducts, err = r.querier.SearchDataproducts(ctx, gensql.SearchDataproductsParams{Query: query, Limit: int32(limit), Offset: int32(offset)})
	}
	if err != nil {
		return nil, err
	}
	for _, r := range dataproducts {
		results = append(results, &openapi.SearchResultEntry{
			Id:      r.ID.String(),
			Name:    r.Name,
			Type:    openapi.SearchResultTypeDataproduct,
			Excerpt: makeExcerpt(r.Description),
			Url:     "/api/dataproducts/" + r.ID.String(),
		})
	}

	// If query is empty, the search result is empty. So we do a regular SELECT * instead
	if query == "" {
		datasets, err = r.querier.GetDatasets(ctx, gensql.GetDatasetsParams{Limit: int32(limit), Offset: int32(offset)})
	} else {
		datasets, err = r.querier.SearchDatasets(ctx, gensql.SearchDatasetsParams{Query: query, Limit: int32(limit), Offset: int32(offset)})
	}
	if err != nil {
		return nil, err
	}
	for _, r := range datasets {
		results = append(results, &openapi.SearchResultEntry{
			Id:      r.ID.String(),
			Name:    r.Name,
			Type:    openapi.SearchResultTypeDataproduct,
			Excerpt: makeExcerpt(r.Description),
			Url:     "/api/datasets/" + r.ID.String(),
		})
	}

	return results, nil
}

func dataproductFromSQL(dataproduct gensql.Dataproduct) *openapi.Dataproduct {
	return &openapi.Dataproduct{
		Id:           dataproduct.ID.String(),
		Name:         dataproduct.Name,
		Created:      dataproduct.Created,
		LastModified: dataproduct.LastModified,
		Description:  nullStringToPtr(dataproduct.Description),
		Keywords:     &dataproduct.Keywords,
		Owner: openapi.Owner{
			Team: dataproduct.Team,
		},
		Repo: nullStringToPtr(dataproduct.Repo),
		Slug: dataproduct.Slug,
	}
}

func datasetFromSQL(dataset gensql.Dataset) *openapi.Dataset {
	return &openapi.Dataset{
		Id:            dataset.ID.String(),
		DataproductId: dataset.DataproductID.String(),
		Name:          dataset.Name,
		Description:   nullStringToPtr(dataset.Description),
		Pii:           dataset.Pii,
		Bigquery: openapi.BigQuery{
			ProjectId: dataset.ProjectID,
			Dataset:   dataset.Dataset,
			Table:     dataset.TableName,
		},
	}
}
