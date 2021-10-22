//go:build integration_test

package e2etests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFiles(t *testing.T) {
	// These tests are defined in testdata/ with the format:
	//
	// [graphql query]
	// RETURNS
	// json response
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) != ".txt" {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			q, expected, err := splitTestFile(f.Name())
			if err != nil {
				t.Fatal(err)
			}
			val, err := doQuery(q)
			if err != nil {
				t.Fatal(err)
			}

			if !cmp.Equal(val, expected, cmp.FilterPath(ignoreID, cmp.Ignore())) {
				t.Error(cmp.Diff(val, expected, cmp.FilterPath(ignoreID, cmp.Ignore())))
			}
		})
	}
}

func splitTestFile(fname string) (q string, expected map[string]interface{}, err error) {
	b, err := ioutil.ReadFile("testdata/" + fname)
	if err != nil {
		return "", nil, err
	}

	parts := bytes.Split(b, []byte("RETURNS"))
	q = string(parts[0])
	if err := json.Unmarshal(parts[1], &expected); err != nil {
		return "", nil, err
	}

	return q, expected, nil
}

func ignoreID(p cmp.Path) bool {
	return p.Last().String() == `["id"]`
}

func doQuery(q string) (map[string]interface{}, error) {
	b, err := json.Marshal(struct {
		OperationName *string                `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
		Query         string                 `json:"query"`
	}{nil, map[string]interface{}{}, q})
	if err != nil {
		return nil, err
	}

	resp, err := server.Client().Post(server.URL+"/api/query", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ret := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, err
	}

	return ret, nil
}
