package metabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Client struct {
	c         *http.Client
	password  string
	url       string
	username  string
	expiry    time.Time
	sessionID string
}

func NewClient(url, username, password string) *Client {
	return &Client{
		c:        http.DefaultClient,
		url:      url,
		password: password,
		username: username,
	}
}

func (c *Client) request(ctx context.Context, method, path string, body interface{}, v interface{}) error {
	err := c.ensureValidSession(ctx)
	if err != nil {
		return err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url+path, buf)
	if err != nil {
		return err
	}

	req.Header.Set("X-Metabase-Session", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.c.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		_, err := io.Copy(os.Stdout, res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("non 2xx status code, got: %v", res.StatusCode)
	}

	if v == nil {
		return nil
	}

	return json.NewDecoder(res.Body).Decode(v)
}

func (c *Client) ensureValidSession(ctx context.Context) error {
	if c.sessionID != "" && c.expiry.After(time.Now()) {
		return nil
	}

	payload := fmt.Sprintf(`{"username": "%v", "password": "%v"}`, c.username, c.password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/session", strings.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := c.c.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("not statuscode 200 OK when creating session, got: %v: %v", res.StatusCode, string(b))
	}

	var session struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return err
	}

	c.sessionID = session.ID
	c.expiry = time.Now().Add(13 * 24 * time.Hour)
	return nil
}

type Database struct {
	ID        int
	DatasetID string
	ProjectID string
	NadaID    string
}

func (c *Client) Databases(ctx context.Context) ([]Database, error) {
	v := struct {
		Data []struct {
			Details struct {
				DatasetID string `json:"dataset-id"`
				ProjectID string `json:"project-id"`
				NadaID    string `json:"nada-id"`
			} `json:"details"`
			ID int `json:"id"`
		} `json:"data"`
	}{}

	if err := c.request(ctx, http.MethodGet, "/database", nil, &v); err != nil {
		return nil, err
	}

	ret := []Database{}
	for _, db := range v.Data {
		ret = append(ret, Database{
			ID:        db.ID,
			DatasetID: db.Details.DatasetID,
			ProjectID: db.Details.ProjectID,
			NadaID:    db.Details.NadaID,
		})
	}

	return ret, nil
}

type NewDatabase struct {
	AutoRunQueries bool    `json:"auto_run_queries"`
	Details        Details `json:"details"`
	Engine         string  `json:"engine"`
	IsFullSync     bool    `json:"is_full_sync"`
	Name           string  `json:"name"`
}

type Details struct {
	DatasetID          string `json:"dataset-id"`
	ProjectID          string `json:"project-id"`
	ServiceAccountJSON string `json:"service-account-json"`
	NadaID             string `json:"nada-id"`
}

func (c *Client) CreateDatabase(ctx context.Context, name, saJSON string, ds *models.BigQuery) (string, error) {
	db := NewDatabase{
		Name: name,
		Details: Details{
			DatasetID:          ds.Dataset,
			NadaID:             ds.DataproductID.String(),
			ProjectID:          ds.ProjectID,
			ServiceAccountJSON: saJSON,
		},
		Engine:         "bigquery-cloud-sdk",
		IsFullSync:     true,
		AutoRunQueries: true,
	}
	var v struct {
		ID int `json:"id"`
	}
	err := c.request(ctx, http.MethodPost, "/database", db, &v)
	return strconv.Itoa(v.ID), err
}

func (c *Client) HideTables(ctx context.Context, ids []int) error {
	t := struct {
		Ids            []int  `json:"ids"`
		VisibilityType string `json:"visibility_type"`
	}{
		Ids:            ids,
		VisibilityType: "hidden",
	}
	return c.request(ctx, http.MethodPut, "/table", t, nil)
}

type Field struct {
}

type Table struct {
	Name   string `json:"name"`
	ID     int    `json:"id"`
	Fields []struct {
		DatabaseType string `json:"database_type"`
		ID           int    `json:"id"`
	} `json:"fields"`
}

func (c *Client) Tables(ctx context.Context, dbID string) ([]Table, error) {
	var v struct {
		Tables []Table `json:"tables"`
	}

	if err := c.request(ctx, http.MethodGet, "/database/"+dbID+"/metadata", nil, &v); err != nil {
		return nil, err
	}

	ret := []Table{}
	for _, t := range v.Tables {
		ret = append(ret, Table{
			Name:   t.Name,
			ID:     t.ID,
			Fields: t.Fields,
		})
	}
	return ret, nil

}

func (c *Client) DeleteDatabase(ctx context.Context, id string) error {
	return c.request(ctx, http.MethodDelete, "/database/"+id, nil, nil)
}

func (c *Client) AutoMapSemanticTypes(ctx context.Context, dbID string) error {
	tables, err := c.Tables(ctx, dbID)
	if err != nil {
		return err
	}

	for _, t := range tables {
		for _, f := range t.Fields {
			if f.DatabaseType == "STRING" {
				if err := c.MapSemanticType(ctx, f.ID); err != nil {
					return err
				}
				fmt.Println("Mapped semantic type for field", f.ID)
			}
		}
	}
	return nil
}

func (c *Client) MapSemanticType(ctx context.Context, fieldID int) error {
	payload := strings.NewReader(`{"semantic_type": "type/Name"}`)
	return c.request(ctx, http.MethodPut, "/field/"+strconv.Itoa(fieldID), payload, nil)
}
