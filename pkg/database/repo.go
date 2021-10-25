package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/tabbed/pqtype"

	"github.com/google/uuid"

	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
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
	log     *logrus.Entry
}

type Querier interface {
	gensql.Querier
	WithTx(tx *sql.Tx) *gensql.Queries
}

func New(dbConnDSN string, log *logrus.Entry) (*Repo, error) {
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
		log:     log,
	}, nil
}

func slugify(maybeslug *string, fallback string) string {
	if maybeslug != nil {
		return *maybeslug
	}
	// TODO(thokra): Smartify this?
	return url.PathEscape(fallback)
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

func (r *Repo) GetDataproducts(ctx context.Context, limit, offset int) ([]*models.Dataproduct, error) {
	datasets := []*models.Dataproduct{}

	res, err := r.querier.GetDataproducts(ctx, gensql.GetDataproductsParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, fmt.Errorf("getting datasets from database: %w", err)
	}

	for _, entry := range res {
		datasets = append(datasets, dataproductFromSQL(entry))
	}

	return datasets, nil
}

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

func (r *Repo) GetCollection(ctx context.Context, id uuid.UUID) (*models.Collection, error) {
	res, err := r.querier.GetCollection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	col := collectionFromSQL(res)

	return col, nil
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

func (r *Repo) DeleteDataproduct(ctx context.Context, id uuid.UUID) error {
	if err := r.querier.DeleteDataproduct(ctx, id); err != nil {
		return fmt.Errorf("deleting dataproduct from database: %w", err)
	}

	return nil
}

func (r *Repo) GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error) {
	return r.querier.GetBigqueryDatasources(ctx)
}

func (r *Repo) GetBigqueryDatasource(ctx context.Context, dataproductID uuid.UUID) (models.BigQuery, error) {
	bq, err := r.querier.GetBigqueryDatasource(ctx, dataproductID)
	if err != nil {
		return models.BigQuery{}, err
	}

	return models.BigQuery{
		ProjectID: bq.ProjectID,
		Dataset:   bq.Dataset,
		Table:     bq.TableName,
	}, nil
}

func (r *Repo) UpdateCollection(ctx context.Context, id uuid.UUID, new models.UpdateCollection) (*models.Collection, error) {
	var keywords []string
	if new.Keywords != nil {
		keywords = []string{}
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

func (r *Repo) CreateDataproduct(ctx context.Context, dp models.NewDataproduct) (*models.Dataproduct, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	if dp.Keywords == nil {
		dp.Keywords = []string{}
	}

	querier := r.querier.WithTx(tx)
	created, err := querier.CreateDataproduct(ctx, gensql.CreateDataproductParams{
		Name:        dp.Name,
		Description: ptrToNullString(dp.Description),
		Pii:         dp.Pii,
		Type:        "bigquery",
		OwnerGroup:  dp.Group,
		Slug:        slugify(dp.Slug, dp.Name),
		Repo:        ptrToNullString(dp.Repo),
		Keywords:    dp.Keywords,
	})
	if err != nil {
		return nil, err
	}

	_, err = querier.CreateBigqueryDatasource(ctx, gensql.CreateBigqueryDatasourceParams{
		DataproductID: created.ID,
		ProjectID:     dp.BigQuery.ProjectID,
		Dataset:       dp.BigQuery.Dataset,
		TableName:     dp.BigQuery.Table,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back dataproduct and datasource_bigquery transaction")
		}
		return nil, err
	}

	for _, subj := range dp.Requesters {
		err = querier.CreateDataproductRequester(ctx, gensql.CreateDataproductRequesterParams{
			DataproductID: created.ID,
			Subject:       subj,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				r.log.WithError(err).Error("Rolling back dataproduct and datasource_bigquery transaction")
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	ret := dataproductFromSQL(created)
	return ret, nil
}

func (r *Repo) GetDataproduct(ctx context.Context, id uuid.UUID) (*models.Dataproduct, error) {
	res, err := r.querier.GetDataproduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting dataproduct from database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) UpdateDataproduct(ctx context.Context, id uuid.UUID, new models.UpdateDataproduct) (*models.Dataproduct, error) {
	if new.Keywords == nil {
		new.Keywords = []string{}
	}
	res, err := r.querier.UpdateDataproduct(ctx, gensql.UpdateDataproductParams{
		Name:        new.Name,
		Description: ptrToNullString(new.Description),
		ID:          id,
		Pii:         new.Pii,
		Slug:        slugify(new.Slug, new.Name),
		Repo:        ptrToNullString(new.Repo),
		Keywords:    new.Keywords,
	})
	if err != nil {
		return nil, fmt.Errorf("updating dataproduct in database: %w", err)
	}

	return dataproductFromSQL(res), nil
}

func (r *Repo) AddRequesterToDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) error {
	return r.querier.CreateDataproductRequester(ctx, gensql.CreateDataproductRequesterParams{
		DataproductID: dataproductID,
		Subject:       subject,
	})
}

func (r *Repo) RemoveRequesterFromDataproduct(ctx context.Context, dataproductID uuid.UUID, subject string) error {
	return r.querier.DeleteDataproductRequester(ctx, gensql.DeleteDataproductRequesterParams{
		DataproductID: dataproductID,
		Subject:       subject,
	})
}

func (r *Repo) ListAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID) ([]*models.Access, error) {
	access, err := r.querier.ListAccessToDataproduct(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	ret := []*models.Access{}
	for _, a := range access {
		ret = append(ret, accessFromSQL(a))
	}

	return ret, nil
}

func (r *Repo) GrantAccessToDataproduct(ctx context.Context, dataproductID uuid.UUID, expires *time.Time, subject, granter string) (*models.Access, error) {
	fmt.Println("EXPIRES", expires)
	access, err := r.querier.GrantAccessToDataproduct(ctx, gensql.GrantAccessToDataproductParams{
		DataproductID: dataproductID,
		Subject:       subject,
		Expires:       ptrToNullTime(expires),
		Granter:       granter,
	})
	if err != nil {
		return nil, err
	}

	return accessFromSQL(access), nil
}

func (r *Repo) GetAccessToDataproduct(ctx context.Context, id uuid.UUID) (*models.Access, error) {
	access, err := r.querier.GetAccessToDataproduct(ctx, id)
	if err != nil {
		return nil, err
	}
	return accessFromSQL(access), nil
}

func (r *Repo) RevokeAccessToDataproduct(ctx context.Context, id uuid.UUID) error {
	return r.querier.RevokeAccessToDataproduct(ctx, id)
}

func (r *Repo) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	if err := r.querier.DeleteCollection(ctx, id); err != nil {
		return fmt.Errorf("deleting dataproduct_collection from database: %w", err)
	}

	return nil
}

func (r *Repo) GetDataproductRequesters(ctx context.Context, id uuid.UUID) ([]string, error) {
	return r.querier.GetDataproductRequesters(ctx, id)
}

func (r *Repo) GetDataproductMetadata(ctx context.Context, id uuid.UUID) (*models.TableMetadata, error) {
	ds, err := r.querier.GetBigqueryDatasource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting bigquery datasource from database: %w", err)
	}

	var schema []models.TableColumn
	if ds.Schema.Valid {
		if err := json.Unmarshal(ds.Schema.RawMessage, &schema); err != nil {
			return nil, fmt.Errorf("unmarshalling schema: %w", err)
		}
	}

	return &models.TableMetadata{
		ID:     ds.DataproductID,
		Schema: schema,
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

func (r *Repo) Search(ctx context.Context, query *models.SearchQuery) ([]models.SearchResult, error) {
	res, err := r.querier.Search(ctx, gensql.SearchParams{
		Query:   ptrToString(query.Text),
		Keyword: ptrToString(query.Keyword),
	})
	if err != nil {
		return nil, err
	}
	ranks := map[string]float32{}
	var dataproducts []uuid.UUID
	var collections []uuid.UUID
	for _, sr := range res {
		switch sr.ElementType {
		case "dataproduct":
			dataproducts = append(dataproducts, sr.ElementID)
		case "collection":
			collections = append(collections, sr.ElementID)
		}
		ranks[sr.ElementType+sr.ElementID.String()] = sr.TsRankCd
	}

	dps, err := r.querier.GetDataproductsByIDs(ctx, dataproducts)
	if err != nil {
		return nil, err
	}
	cols, err := r.querier.GetCollectionsByIDs(ctx, collections)
	if err != nil {
		return nil, err
	}

	ret := []models.SearchResult{}
	for _, d := range dps {
		ret = append(ret, dataproductFromSQL(d))
	}
	for _, c := range cols {
		ret = append(ret, collectionFromSQL(c))
	}

	getRank := func(m models.SearchResult) float32 {
		switch m := m.(type) {
		case *models.Dataproduct:
			return ranks["dataproduct"+m.ID.String()]
		case *models.Collection:
			return ranks["collection"+m.ID.String()]
		default:
			return -1
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return getRank(ret[i]) < getRank(ret[j])
	})

	return ret, nil
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

func dataproductFromSQL(dp gensql.Dataproduct) *models.Dataproduct {
	return &models.Dataproduct{
		ID:           dp.ID,
		Name:         dp.Name,
		Created:      dp.Created,
		LastModified: dp.LastModified,
		Description:  nullStringToPtr(dp.Description),
		Slug:         dp.Slug,
		Repo:         nullStringToPtr(dp.Repo),
		Pii:          dp.Pii,
		Keywords:     dp.Keywords,
		Owner: &models.Owner{
			Group: dp.Group,
		},
		Type: dp.Type,
	}
}

func accessFromSQL(access gensql.DataproductAccess) *models.Access {
	return &models.Access{
		ID:            access.ID,
		Subject:       access.Subject,
		Granter:       access.Granter,
		Expires:       nullTimeToPtr(access.Expires),
		Created:       access.Created,
		Revoked:       nullTimeToPtr(access.Revoked),
		DataproductID: access.DataproductID,
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

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
