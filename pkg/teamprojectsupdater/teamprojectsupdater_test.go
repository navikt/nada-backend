package teamprojectsupdater_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/navikt/nada-backend/pkg/config"
	"github.com/navikt/nada-backend/pkg/teamprojectsupdater"
	"github.com/stretchr/testify/assert"
)

func TestTeamProjectsUpdater(t *testing.T) {
	teamProjects := make(map[string][]string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		file, err := ioutil.ReadFile(fmt.Sprintf("testdata/%v", request.URL.Path))
		assert.NoError(t, err)
		fmt.Fprintln(writer, string(file))
	}))

	cfg := config.Config{
		DevTeamProjectsOutputURL:  server.URL + "/prod-output.json", // filename in ./testdata
		ProdTeamProjectsOutputURL: server.URL + "/dev-output.json",  // filename in ./testdata
	}

	err := teamprojectsupdater.FetchTeamGoogleProjectsMapping(ctx, teamProjects, cfg)

	assert.NoError(t, err)

	assert.Equal(t, 3, len(teamProjects))
	assert.Equal(t, 2, len(teamProjects["team-a"]))
	assert.Equal(t, 2, len(teamProjects["team-b"]))
	assert.Equal(t, 1, len(teamProjects["team-c"]))
}
