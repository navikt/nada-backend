package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/tabbed/pqtype"

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
	querier Querier
	db      *sql.DB
}

type Querier interface {
	gensql.Querier
	WithTx(tx *sql.Tx) *gensql.Queries
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
		db:      db,
	}, nil
}

func slugify(maybeslug *string, fallback string) string {
	if maybeslug != nil {
		return *maybeslug
	}
	// TODO(thokra): Smartify this?
	return url.PathEscape(fallback)
}

func (r *Repo) CreateCollection(ctx context.Context, dp openapi.NewCollection) (*openapi.Collection, error) {
	var keywords []string
	if dp.Keywords != nil {
		keywords = *dp.Keywords
	}
	res, err := r.querier.CreateCollection(ctx, gensql.CreateCollectionParams{
		Name:        dp.Name,
		Description: ptrToNullString(dp.Description),
		Slug:        slugify(dp.Slug, dp.Name),
		Repo:        ptrToNullString(dp.Repo),
		OwnerGroup:  dp.Owner.Group,
		Keywords:    keywords,
	})
	if err != nil {
		return nil, err
	}

	return collectionFromSQL(res), nil
}

func (r *Repo) GetDataproducts(ctx context.Context, limit, offset int) ([]*openapi.Dataproduct, error) {
	datasets := []*openapi.Dataproduct{}

	res, err := r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting datasets from database: %w", err)
	}

	for _, entry := range res {
		datasets = append(datasets, dataproductFromSQL(entry))
	}

	return datasets, nil
}

func (r *Repo) GetCollections(ctx context.Context, limit, offset int) ([]*openapi.Collection, error) {
	dataproducts := []*openapi.Collection{}

	res, err := r.querier.GetCollections(ctx, gensql.GetCollectionsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting dataproducts from database: %w", err)
	}

	for _, entry := range res {
		dataproduct := collectionFromSQL(entry)
		dataproducts = append(dataproducts, dataproduct)
	}

	return dataproducts, nil
}

func (r *Repo) GetCollection(ctx context.Context, id string) (*openapi.Collection, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}
	res, err := r.querier.GetCollection(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	dp := collectionFromSQL(res)

	colElems, err := r.querier.GetCollectionElements(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting collection elements: %w", err)
	}

	for _, elem := range colElems {
		dp.Elements = append(dp.Elements, openapi.CollectionElement{
			ElementId:   elem.ElementID,
			ElementType: openapi.CollectionElementType(elem.ElementType),
		})
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

func (r *Repo) GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error) {
	return r.querier.GetBigqueryDatasources(ctx)
}

func (r *Repo) UpdateCollection(ctx context.Context, id string, new openapi.UpdateCollection) (*openapi.Collection, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	var keywords []string
	if new.Keywords != nil {
		keywords = *new.Keywords
	}

	res, err := r.querier.UpdateCollection(ctx, gensql.UpdateCollectionParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		ID:          uid,
		Keywords:    keywords,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return collectionFromSQL(res), nil
}

func (r *Repo) CreateDataproduct(ctx context.Context, dp openapi.NewDataproduct) (*openapi.Dataproduct, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	querier := r.querier.WithTx(tx)

	keywords := []string{}
	if dp.Keywords != nil && *dp.Keywords != nil {
		keywords = *dp.Keywords
	}
	created, err := querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:        dp.Name,
		Description: ptrToNullString(dp.Description),
		Pii:         dp.Pii,
		Type:        "bigquery",
		OwnerGroup:  dp.Owner.Group,
		Slug:        ptrToNullString(dp.Slug),
		Repo:        ptrToNullString(dp.Repo),
		Keywords:    keywords,
	})
	if err != nil {
		return nil, err
	}

	datasource, err := MapDatasource(dp.Datasource)
	if err != nil {
		return nil, err
	}

	_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
		DataproductID: created.ID,
		ProjectID:     datasource.ProjectId,
		Dataset:       datasource.Dataset,
		TableName:     datasource.Table,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			log.WithError(err).Error("Rolling back dataproduct and datasource_bigquery transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	ret := dataproductFromSQL(created)
	ret.Datasource = datasource
	return ret, nil
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

	bq, err := r.querier.GetBigqueryDatasource(ctx, res.ID)
	if err != nil {
		return nil, fmt.Errorf("getting bigquery datasource from database: %w", err)
	}

	ret := dataproductFromSQL(res)
	ret.Datasource = openapi.Bigquery{
		Dataset:   bq.Dataset,
		ProjectId: bq.ProjectID,
		Table:     bq.TableName,
	}

	return ret, nil
}

func (r *Repo) UpdateDataproduct(ctx context.Context, id string, new openapi.UpdateDataproduct) (*openapi.Dataproduct, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	keywords := []string{}
	if new.Keywords != nil && *new.Keywords != nil {
		keywords = *new.Keywords
	}
	res, err := r.querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		ID:          uid,
		Pii:         new.Pii,
		Slug:        ptrToNullString(new.Slug),
		Repo:        ptrToNullString(new.Repo),
		Keywords:    keywords,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) DeleteCollection(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parsing uuid: %w", err)
	}

	if err := r.querier.DeleteCollection(ctx, uid); err != nil {
		return fmt.Errorf("deleting dataproduct_collection from database: %w", err)
	}

	return nil
}

func (r *Repo) GetDataproductMetadata(ctx context.Context, id string) (*openapi.DataproductMetadata, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parsing uuid: %w", err)
	}

	ds, err := r.querier.GetBigqueryDatasource(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("getting bigquery datasource from database: %w", err)
	}

	var schema []openapi.TableColumn
	if ds.Schema.Valid {
		if err := json.Unmarshal(ds.Schema.RawMessage, &schema); err != nil {
			return nil, fmt.Errorf("unmarshalling schema: %w", err)
		}
	}

	return &openapi.DataproductMetadata{
		DataproductId: ds.DataproductID.String(),
		Schema:        schema,
	}, nil
}

func (r *Repo) UpdateBigqueryDatasource(ctx context.Context, id uuid.UUID, schema json.RawMessage) error {
	err := r.querier.UpdateBigqueryDatasourceSchema(ctx, gensql.UpdateBigqueryDatasourceSchemaParams{
		DataproductID: id,
		Schema: pqtype.NullRawMessage{
			RawMessage: schema,
			Valid:      true,
		},
	})
	if err != nil {
		return fmt.Errorf("updating datasource_bigquery schema: %w", err)
	}

	return nil
}

func (r *Repo) AddToCollection(ctx context.Context, collectionID string, body openapi.CollectionElement) error {
	return r.querier.CreateCollectionElement(ctx, gensql.CreateCollectionElementParams{
		ElementID:    body.ElementId,
		ElementType:  string(body.ElementType),
		CollectionID: collectionID,
	})
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
		collections  []gensql.Collection
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
		collections, err = r.querier.GetCollections(ctx, gensql.GetCollectionsParams{Limit: int32(limit), Offset: int32(offset)})
	} else {
		collections, err = r.querier.SearchCollections(ctx, gensql.SearchCollectionsParams{Query: query, Limit: int32(limit), Offset: int32(offset)})
	}
	if err != nil {
		return nil, err
	}
	for _, r := range collections {
		results = append(results, &openapi.SearchResultEntry{
			Id:      r.ID.String(),
			Name:    r.Name,
			Type:    openapi.SearchResultTypeCollection,
			Excerpt: makeExcerpt(r.Description),
			Url:     "/api/collections/" + r.ID.String(),
		})
	}

	return results, nil
}

func collectionFromSQL(dataproduct gensql.Collection) *openapi.Collection {
	return &openapi.Collection{
		Id:           dataproduct.ID.String(),
		Name:         dataproduct.Name,
		Created:      dataproduct.Created,
		LastModified: dataproduct.LastModified,
		Description:  nullStringToPtr(dataproduct.Description),
		Keywords:     &dataproduct.Keywords,
		Owner: openapi.Owner{
			Group: dataproduct.Group,
		},
		Repo: nullStringToPtr(dataproduct.Repo),
		Slug: dataproduct.Slug,
	}
}

func dataproductFromSQL(dataset gensql.Dataproduct) *openapi.Dataproduct {
	return &openapi.Dataproduct{
		Id:          dataset.ID.String(),
		Name:        dataset.Name,
		Description: nullStringToPtr(dataset.Description),
		Pii:         dataset.Pii,
		Keywords:    dataset.Keywords,
		Repo:        nullStringToPtr(dataset.Repo),
		Slug:        nullStringToPtr(dataset.Slug),
		Type:        openapi.DataproductType(dataset.Type),
		Owner: openapi.Owner{
			Group: dataset.Group,
		},
	}
}

func MapDatasource(source openapi.Datasource) (openapi.Bigquery, error) {
	b, err := json.Marshal(source)
	if err != nil {
		return openapi.Bigquery{}, err
	}

	var ds openapi.Bigquery
	if err := json.Unmarshal(b, &ds); err != nil {
		return ds, err
	}
	return ds, nil
}
