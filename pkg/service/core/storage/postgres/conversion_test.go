package postgres_test

import (
	"testing"

	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
)

type testFrom struct {
	data string
}

type testTo struct {
	data string
}

func (t *testFrom) To() (*testTo, error) {
	return &testTo{data: t.data}, nil
}

func TestConversion(t *testing.T) {
	from := &testFrom{data: "test"}

	to, err := postgres.From(from)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if to.data != "test" {
		t.Errorf("unexpected data: %v", to.data)
	}
}
