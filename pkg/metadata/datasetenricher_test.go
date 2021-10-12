package metadata

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
)

func TestDatasetEnricher(t *testing.T) {
	datastorer := &metadatastorer{
		datasets: []*openapi.Dataset{
			{Id: "dataset_1"},
			{Id: "dataset_2"},
			{Id: "dataset_3"},
			{Id: "dataset_4"},
		},
	}
	datacatalog := &datacatalogMock{}
	log := logrus.Logger{Out: ioutil.Discard}
	enricher := New(datacatalog, datastorer, log.WithField("", ""))

	if err := enricher.syncMetadata(context.Background()); err != nil {
		t.Fatal(err)
	}

	expectedUpdates := []string{"dataset_1", "dataset_2", "dataset_3", "dataset_4"}
	if !cmp.Equal(datastorer.writtenIDs, expectedUpdates) {
		t.Error(cmp.Diff(datastorer.writtenIDs, expectedUpdates))
	}
}

type datacatalogMock struct {
	schema Schema
	err    error
}

func (d *datacatalogMock) GetDatasetSchema(ctx context.Context, ds openapi.BigQuery) (Schema, error) {
	if d.err != nil {
		return Schema{}, d.err
	}
	return d.schema, nil
}

type metadatastorer struct {
	datasets   []*openapi.Dataset
	writtenIDs []string
}

func (m *metadatastorer) GetDatasets(ctx context.Context, limit int, offset int) ([]*openapi.Dataset, error) {
	return m.datasets, nil
}

func (m *metadatastorer) WriteDatasetMetadata(ctx context.Context, datasetID string, schema json.RawMessage) error {
	m.writtenIDs = append(m.writtenIDs, datasetID)
	return nil
}
