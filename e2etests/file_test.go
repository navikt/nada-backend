//go:build integration_test

package e2etests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

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
		if f.IsDir() {
			nested, err := os.ReadDir(filepath.Join("testdata", f.Name()))
			if err != nil {
				t.Fatal(err)
			}

			state := &state{
				data: map[string]interface{}{},
			}
			for _, nf := range nested {
				if filepath.Ext(nf.Name()) != ".txt" {
					continue
				}
				testFile(t, state, filepath.Join(f.Name(), nf.Name()))
			}
		}
		if filepath.Ext(f.Name()) != ".txt" {
			continue
		}
		testFile(t, &state{}, f.Name())
	}
}

func testFile(t *testing.T, state *state, fname string) {
	t.Run(fname, func(t *testing.T) {
		q, expected, store, err := splitTestFile(fname)
		if err != nil {
			t.Fatal(err)
		}
		val, err := doQuery(state, q, store)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(val, expected, cmp.FilterPath(ignoreID, cmp.Ignore())) {
			t.Error(cmp.Diff(val, expected, cmp.FilterPath(ignoreID, cmp.Ignore())))
		}
	})
}

type storeRequest struct {
	key  string
	path string
}

func splitTestFile(fname string) (q string, expected map[string]interface{}, store []storeRequest, err error) {
	b, err := ioutil.ReadFile("testdata/" + fname)
	if err != nil {
		return "", nil, nil, err
	}

	parts := bytes.SplitN(b, []byte("RETURNS"), 2)
	q = string(parts[0])

	srs := bytes.Split(parts[1], []byte("STORE"))
	if err := json.Unmarshal(srs[0], &expected); err != nil {
		return "", nil, nil, err
	}

	if len(srs) > 1 {
		for _, s := range srs[1:] {
			sp := strings.Split(strings.TrimSpace(string(s)), "=")
			store = append(store, storeRequest{
				key:  strings.TrimSpace(sp[0]),
				path: strings.TrimSpace(sp[1]),
			})
		}
	}

	return q, expected, store, nil
}

func ignoreID(p cmp.Path) bool {
	return p.Last().String() == `["id"]`
}

type state struct {
	data map[string]interface{}
}

func doQuery(state *state, q string, store []storeRequest) (map[string]interface{}, error) {
	tpl, err := template.New("query").Parse(q)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, state.data); err != nil {
		return nil, err
	}

	b, err := json.Marshal(
		struct {
			OperationName *string                `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
			Query         string                 `json:"query"`
		}{
			nil,
			map[string]interface{}{},
			buf.String(),
		},
	)
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

	for _, s := range store {
		var (
			root = ret
			val  interface{}
		)
		pathParts := strings.Split(s.path, ".")

		for i, kp := range pathParts {
			if i == len(pathParts)-1 {
				// Last element of pathParts
				val = root[kp]
				break
			}
			root = root[kp].(map[string]interface{})
		}

		state.data[s.key] = val
	}

	return ret, nil
}
