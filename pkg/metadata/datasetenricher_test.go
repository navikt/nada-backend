package metadata

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/sirupsen/logrus"
)

func TestDatasetEnricher(t *testing.T) {
	expectedUpdates := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	datastorer := &metadatastorer{
		ds: []gensql.DatasourceBigquery{
			{DataproductID: expectedUpdates[0]},
			{DataproductID: expectedUpdates[1]},
			{DataproductID: expectedUpdates[2]},
			{DataproductID: expectedUpdates[3]},
		},
	}
	datacatalog := &datacatalogMock{}
	log := logrus.Logger{Out: ioutil.Discard}
	enricher := New(datacatalog, datastorer, log.WithField("", ""))

	if err := enricher.syncMetadata(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(datastorer.writtenIDs, expectedUpdates) {
		t.Error(cmp.Diff(datastorer.writtenIDs, expectedUpdates))
	}
}

type datacatalogMock struct {
	schema Schema
	err    error
}

func (d *datacatalogMock) GetDatasetSchema(ctx context.Context, ds gensql.DatasourceBigquery) (Schema, error) {
	if d.err != nil {
		return Schema{}, d.err
	}
	return d.schema, nil
}

type metadatastorer struct {
	ds         []gensql.DatasourceBigquery
	writtenIDs []uuid.UUID
}

func (m *metadatastorer) GetBigqueryDatasources(ctx context.Context) ([]gensql.DatasourceBigquery, error) {
	return m.ds, nil
}

func (m *metadatastorer) UpdateBigqueryDatasource(ctx context.Context, id uuid.UUID, schema json.RawMessage) error {
	m.writtenIDs = append(m.writtenIDs, id)
	return nil
}
