//go:build integration_test

package integration

import (
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core"
	"github.com/navikt/nada-backend/pkg/service/core/api/gcp"
	httpapi "github.com/navikt/nada-backend/pkg/service/core/api/http"
	"github.com/navikt/nada-backend/pkg/service/core/handlers"
	"github.com/navikt/nada-backend/pkg/service/core/routes"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	testTeam    = "team"
	storyBucket = "stories"
	defaultHtml = "<html><h1>Story</h1></html>"
)

func TestStory(t *testing.T) {
	ctx := context.Background()

	storyID, err := prepareStoryTests(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get story", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL + "/quarto/" + storyID.String() + "/index.html")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
		}

		respb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(respb) != defaultHtml {
			t.Fatalf("expected object read to be %v, got %v", defaultHtml, string(respb))
		}
	})

	t.Run("get story with redirect", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL + "/quarto/" + storyID.String())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
		}

		respb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(respb) != defaultHtml {
			t.Fatalf("expected object read to be %v, got %v", defaultHtml, string(respb))
		}
	})

	newHtml := "<html><h1>Quarto updated</h1></html>"

	teamToken, err := service.GetNadaToken(ctx, testTeam)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update story", func(t *testing.T) {
		body, contentType, err := createMultipartForm(newHtml)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/quarto/update/"+storyID.String(), body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Bearer "+teamToken.String())
		req.Header.Add("Content-Type", contentType)

		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("get story after update", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL + "/quarto/" + storyID.String() + "/index.html")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %v, got %v", http.StatusOK, resp.StatusCode)
		}

		respb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(respb) != newHtml {
			t.Fatalf("expected object read to be %v, got %v", newHtml, string(respb))
		}
	})

	t.Run("update story invalid id", func(t *testing.T) {
		body, contentType, err := createMultipartForm(newHtml)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/quarto/update/123", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Bearer "+teamToken.String())
		req.Header.Add("Content-Type", contentType)

		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status code %v, got %v", http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("update story does not exist", func(t *testing.T) {
		nonExistingQuarto := "d7fae699-9852-4367-a136-e6b787e2a5bd"

		body, contentType, err := createMultipartForm(newHtml)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/quarto/update/"+nonExistingQuarto, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Bearer "+teamToken.String())
		req.Header.Add("Content-Type", contentType)

		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected status code %v, got %v", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("update story unauthorized", func(t *testing.T) {
		invalidToken := "d7fae699-9852-4367-a136-e6b787e2a5bd"

		body, contentType, err := createMultipartForm(newHtml)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/quarto/update/"+storyID.String(), body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Bearer "+invalidToken)
		req.Header.Add("Content-Type", contentType)

		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %v, got %v", http.StatusUnauthorized, resp.StatusCode)
		}
	})

	if err := repo.Querier.DeleteNadaToken(ctx, testTeam); err != nil {
		t.Fatal(err)
	}

	t.Run("update story team token not found", func(t *testing.T) {
		teamToken := "d7fae699-9852-4367-a136-e6b787e2a5bd"

		body, contentType, err := createMultipartForm(newHtml)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/quarto/update/"+storyID.String(), body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Bearer "+teamToken)
		req.Header.Add("Content-Type", contentType)

		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %v, got %v", http.StatusInternalServerError, resp.StatusCode)
		}
	})
}

func prepareStoryTests(ctx context.Context) (uuid.UUID, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer client.Close()

	if err := client.Bucket(storyBucket).Create(ctx, "project", nil); err != nil {
		var e *googleapi.Error
		if ok := xerrors.As(err, &e); ok {
			if e.Code != 409 {
				return uuid.UUID{}, err
			}
		}
	}

	description := "this is my story"

	story, err := repo.Querier.CreateStory(ctx, gensql.CreateStoryParams{
		Name:        "story",
		Creator:     "first.last@nav.no",
		Description: description,
		Keywords:    []string{},
		OwnerGroup:  testTeam + "@nav.no",
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	htmlb := []byte(defaultHtml)

	writer := client.Bucket(storyBucket).Object(story.ID.String() + "/index.html").NewWriter(ctx)
	_, err = writer.Write(htmlb)
	if err != nil {
		return uuid.UUID{}, err
	}

	if err = writer.Close(); err != nil {
		return uuid.UUID{}, err
	}

	return story.ID, nil
}

func createMultipartForm(html string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("index.html", "index.html")
	if err != nil {
		return nil, "", err
	}
	_, err = part.Write([]byte(html))
	if err != nil {
		return nil, "", err
	}

	err = writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}
