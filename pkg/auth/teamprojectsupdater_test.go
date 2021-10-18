package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeamProjectsUpdater(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		file, err := ioutil.ReadFile(fmt.Sprintf("testdata/%v", request.URL.Path))
		assert.NoError(t, err)
		fmt.Fprintln(writer, string(file))
	}))

	tup := NewTeamProjectsUpdater(server.URL+"/dev-output.json", server.URL+"/prod-output.json", "token", server.Client())

	err := tup.FetchTeamGoogleProjectsMapping(context.Background())

	assert.NoError(t, err)

	assert.Equal(t, 3, len(tup.teamProjects))
	assert.Equal(t, 2, len(tup.teamProjects["team-a@nav.no"]))
	assert.Equal(t, 2, len(tup.teamProjects["team-b@nav.no"]))
	assert.Equal(t, 1, len(tup.teamProjects["team-c@nav.no"]))
}
