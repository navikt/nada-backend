//go:build integration_test

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/navikt/nada-backend/pkg/auth"
)

func TestTeams_GetGCPProject(t *testing.T) {
	resp, err := client.GetGCPProjects(context.Background(), auth.MockUser.Groups[0].Email)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusNoContent, resp.StatusCode)
	}
}

func TestTeams_GetGCPProject_invalidTeam(t *testing.T) {
	resp, err := client.GetGCPProjects(context.Background(), "invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %v, got %v", http.StatusNoContent, resp.StatusCode)
	}
}
