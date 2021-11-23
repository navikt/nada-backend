package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeamProjectsUpdater(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		file, err := ioutil.ReadFile(fmt.Sprintf("testdata/%v", request.URL.Path))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(writer, string(file))
	}))

	tup := NewTeamProjectsUpdater(server.URL+"/dev-output.json", server.URL+"/prod-output.json", "token", server.Client())

	err := tup.FetchTeamGoogleProjectsMapping(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(tup.teamProjects) != 3 {
		t.Errorf("got: %v, want: %v", len(tup.teamProjects), 3)
	}
	if len(tup.teamProjects["team-a@nav.no"]) != 2 {
		t.Errorf("got: %v, want: %v", tup.teamProjects["team-a@nav.no"], 2)
	}
	if len(tup.teamProjects["team-b@nav.no"]) != 2 {
		t.Errorf("got: %v, want: %v", tup.teamProjects["team-b@nav.no"], 2)
	}
	if len(tup.teamProjects["team-c@nav.no"]) != 1 {
		t.Errorf("got: %v, want: %v", tup.teamProjects["team-c@nav.no"], 1)
	}
}
