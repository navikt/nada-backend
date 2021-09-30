package api

import (
	"testing"

	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/openapi"
	"github.com/sirupsen/logrus"
)

var _ openapi.ServerInterface = (*Server)(nil)

func TestHello(t *testing.T) {
	var _ openapi.ServerInterface = New(&database.Repo{}, &logrus.Entry{})
	t.Log("Hello")
}
