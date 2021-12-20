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
		return fmt.Errorf("%v %v: %w", method, path, err)
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return fmt.Errorf("%v %v: %w", method, path, err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url+path, buf)
	if err != nil {
		return fmt.Errorf("%v %v: %w", method, path, err)
	}

	req.Header.Set("X-Metabase-Session", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.c.Do(req)
	if err != nil {
		return fmt.Errorf("%v %v: %w", method, path, err)
	}

	if res.StatusCode > 299 {
		_, err := io.Copy(os.Stdout, res.Body)
		if err != nil {
			return fmt.Errorf("%v %v: %w", method, path, err)
		}
		return fmt.Errorf("%v %v: non 2xx status code, got: %v", method, path, res.StatusCode)
	}

	if v == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("%v %v: %w", method, path, err)
	}

	return nil
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
	SAEmail   string
}

func (c *Client) Databases(ctx context.Context) ([]Database, error) {
	v := struct {
		Data []struct {
			Details struct {
				DatasetID string `json:"dataset-id"`
				ProjectID string `json:"project-id"`
				NadaID    string `json:"nada-id"`
				SAEmail   string `json:"sa-email"`
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
			SAEmail:   db.Details.SAEmail,
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
	SAEmail            string `json:"sa-email"`
}

func (c *Client) CreateDatabase(ctx context.Context, name, saJSON, saEmail string, ds *models.BigQuery) (int, error) {
	db := NewDatabase{
		Name: name,
		Details: Details{
			DatasetID:          ds.Dataset,
			NadaID:             ds.DataproductID.String(),
			ProjectID:          ds.ProjectID,
			ServiceAccountJSON: saJSON,
			SAEmail:            saEmail,
		},
		Engine:         "bigquery-cloud-sdk",
		IsFullSync:     true,
		AutoRunQueries: true,
	}
	var v struct {
		ID int `json:"id"`
	}
	err := c.request(ctx, http.MethodPost, "/database", db, &v)

	return v.ID, err
}

func (c *Client) HideTables(ctx context.Context, ids []int) error {
	t := struct {
		IDs            []int  `json:"ids"`
		VisibilityType string `json:"visibility_type"`
	}{
		IDs:            ids,
		VisibilityType: "hidden",
	}
	return c.request(ctx, http.MethodPut, "/table", t, nil)
}

type Field struct{}

type Table struct {
	Name   string `json:"name"`
	ID     int    `json:"id"`
	Fields []struct {
		DatabaseType string `json:"database_type"`
		ID           int    `json:"id"`
		SemanticType string `json:"semantic_type"`
	} `json:"fields"`
}

type PermissionGroup struct {
	ID      int                     `json:"id"`
	Name    string                  `json:"name"`
	Members []PermissionGroupMember `json:"members"`
}

type PermissionGroupMember struct {
	ID    int    `json:"membership_id"`
	Email string `json:"email"`
}

type MetabaseUser struct {
	Email string `json:"email"`
	ID    int    `json:"id"`
}

func (c *Client) Tables(ctx context.Context, dbID int) ([]Table, error) {
	var v struct {
		Tables []Table `json:"tables"`
	}

	if err := c.request(ctx, http.MethodGet, fmt.Sprintf("/database/%v/metadata", dbID), nil, &v); err != nil {
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

func (c *Client) DeleteDatabase(ctx context.Context, id int) error {
	return c.request(ctx, http.MethodDelete, fmt.Sprintf("/database/%v", id), nil, nil)
}

func (c *Client) AutoMapSemanticTypes(ctx context.Context, dbID int) error {
	tables, err := c.Tables(ctx, dbID)
	if err != nil {
		return err
	}

	for _, t := range tables {
		for _, f := range t.Fields {
			switch f.DatabaseType {
			case "STRING":
				if err := c.MapSemanticType(ctx, f.ID, "type/Name"); err != nil {
					return err
				}
			case "TIMESTAMP":
				if err := c.MapSemanticType(ctx, f.ID, "type/CreationTimestamp"); err != nil {
					return err
				}
			case "DATE":
				if err := c.MapSemanticType(ctx, f.ID, "type/CreationDate"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Client) MapSemanticType(ctx context.Context, fieldID int, semanticType string) error {
	payload := map[string]string{"semantic_type": semanticType}
	return c.request(ctx, http.MethodPut, "/field/"+strconv.Itoa(fieldID), payload, nil)
}

func (c *Client) CreatePermissionGroup(ctx context.Context, name string) (int, error) {
	group := PermissionGroup{}
	payload := map[string]string{"name": name}
	if err := c.request(ctx, http.MethodPost, "/permissions/group", payload, &group); err != nil {
		return 0, err
	}
	return group.ID, nil
}

func (c *Client) GetPermissionGroup(ctx context.Context, groupID int) ([]PermissionGroupMember, error) {
	g := PermissionGroup{}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("/permissions/group/%v", groupID), nil, &g)
	if err != nil {
		return nil, err
	}

	return g.Members, nil
}

func (c *Client) RemovePermissionGroupMember(ctx context.Context, memberID int) error {
	return c.request(ctx, http.MethodDelete, fmt.Sprintf("/permissions/membership/%v", memberID), nil, nil)
}

func (c *Client) AddPermissionGroupMember(ctx context.Context, groupID int, email string) error {
	var users struct {
		Data []MetabaseUser
	}

	err := c.request(ctx, http.MethodGet, "/user", nil, &users)
	if err != nil {
		return err
	}

	userID, err := getUserID(users.Data, email)
	if err != nil {
		return err
	}

	payload := map[string]int{"group_id": groupID, "user_id": userID}
	return c.request(ctx, http.MethodPost, "/permissions/membership", payload, nil)
}

func (c *Client) RestrictAccessToDatabase(ctx context.Context, groupID, databaseID int) error {
	type permissions struct {
		Native  string `json:"native,omitempty"`
		Schemas string `json:"schemas,omitempty"`
	}
	var permissionGraph struct {
		Groups   map[string]map[string]permissions `json:"groups"`
		Revision int                               `json:"revision"`
	}

	err := c.request(ctx, http.MethodGet, "/permissions/graph", nil, &permissionGraph)
	if err != nil {
		return err
	}

	grpSID := strconv.Itoa(groupID)
	dbSID := strconv.Itoa(databaseID)

	if _, ok := permissionGraph.Groups[grpSID]; !ok {
		permissionGraph.Groups[grpSID] = map[string]permissions{}
	}
	permissionGraph.Groups[grpSID][dbSID] = permissions{Native: "write", Schemas: "all"}

	for gid, permission := range permissionGraph.Groups {
		if gid == "2" {
			// admin group
			continue
		}
		if gid != grpSID {
			permission[dbSID] = permissions{Native: "none", Schemas: "none"}
		}
	}

	if err := c.request(ctx, http.MethodPut, "/permissions/graph", permissionGraph, nil); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeletePermissionGroup(ctx context.Context, groupID int) error {
	return c.request(ctx, http.MethodDelete, fmt.Sprintf("/permissions/group/%v", groupID), nil, nil)
}

func (c *Client) CreateCollection(ctx context.Context, name string) (int, error) {
	collection := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}{
		Name:        name,
		Description: "Collection for " + name,
		Color:       "#509EE3",
	}

	var response struct {
		ID int `json:"id"`
	}

	err := c.request(ctx, http.MethodPost, "/collection", collection, &response)
	if err != nil {
		return 0, err
	}
	return response.ID, nil
}

func (c *Client) SetCollectionAccess(ctx context.Context, groupID, collectionID int) error {
	var cPermissions struct {
		Revision int                          `json:"revision"`
		Groups   map[string]map[string]string `json:"groups"`
	}

	err := c.request(ctx, http.MethodGet, "/collection/graph", nil, &cPermissions)
	if err != nil {
		return err
	}

	sgid := strconv.Itoa(groupID)
	scid := strconv.Itoa(collectionID)
	for gid, permissions := range cPermissions.Groups {
		if gid == "2" {
			continue
		} else if gid == sgid {
			permissions[scid] = "write"
		} else {
			permissions[scid] = "none"
		}
	}

	return c.request(ctx, http.MethodPut, "/collection/graph", cPermissions, nil)
}

func (c *Client) CreateCollectionWithAccess(ctx context.Context, groupID int, name string) (int, error) {
	cid, err := c.CreateCollection(ctx, name)
	if err != nil {
		return 0, err
	}

	if err := c.SetCollectionAccess(ctx, groupID, cid); err != nil {
		return cid, err
	}
	return cid, nil
}

func getUserID(users []MetabaseUser, email string) (int, error) {
	for _, u := range users {
		if u.Email == email {
			return u.ID, nil
		}
	}
	return -1, fmt.Errorf("user %v does not exist in metabase", email)
}
