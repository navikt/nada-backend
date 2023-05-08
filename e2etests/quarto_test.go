//go:build integration_test

package e2etests

import (
	"bytes"
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"golang.org/x/xerrors"
	"google.golang.org/api/googleapi"
)

const (
	testTeam     = "team"
	quartoBucket = "quarto"
	defaultHtml  = "<html><h1>Quarto</h1></html>"
)

func TestQuarto(t *testing.T) {
	ctx := context.Background()

	storyID, err := prepareQuartoTests(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get quarto", func(t *testing.T) {
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

	t.Run("get quarto with redirect", func(t *testing.T) {
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

	teamToken, err := repo.GetNadaToken(ctx, testTeam)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update quarto", func(t *testing.T) {
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

	t.Run("get quarto after update", func(t *testing.T) {
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

	t.Run("update quarto invalid id", func(t *testing.T) {
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

	t.Run("update quarto does not exist", func(t *testing.T) {
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

	t.Run("update quarto unauthorized", func(t *testing.T) {
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

	if err := repo.DeleteNadaToken(ctx, testTeam); err != nil {
		t.Fatal(err)
	}

	t.Run("update quarto team token not found", func(t *testing.T) {
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

func prepareQuartoTests(ctx context.Context) (uuid.UUID, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer client.Close()

	if err := client.Bucket(quartoBucket).Create(ctx, "project", nil); err != nil {
		var e *googleapi.Error
		if ok := xerrors.As(err, &e); ok {
			if e.Code != 409 {
				return uuid.UUID{}, err
			}
		}
	}

	description := "this is my quarto"

	story, err := repo.CreateQuartoStory(ctx, "first.last@nav.no", models.NewQuartoStory{
		Name:        "quarto",
		Description: &description,
		Group:       testTeam + "@nav.no",
		Keywords:    []string{},
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	htmlb := []byte(defaultHtml)

	writer := client.Bucket(quartoBucket).Object(story.ID.String() + "/index.html").NewWriter(ctx)
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
