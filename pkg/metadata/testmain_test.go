package metadata

import (
	"context"
	"testing"
)

func TestASFD(t *testing.T) {
	dc, err := NewDatacatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = dc.GetDatasets(context.Background(), "dataplattform-dev-9da3")
	if err != nil {
		t.Fatal(err)
	}
}
