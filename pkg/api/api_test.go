package api

import (
	"testing"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
)

var _ openapi.ServerInterface = (*Server)(nil)

func TestHello(t *testing.T) {
	var _ openapi.ServerInterface = New(&database.Repo{}, &logrus.Entry{})
	t.Log("Hello")
}
