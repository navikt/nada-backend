//go:build integration_test

package e2etests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	NotNullOption = "NOTNULL"
	IgnoreOption  = "IGNORE"
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
		q, expected, store, options, err := splitTestFile(fname, state)
		if err != nil {
			t.Fatal(err)
		}
		val, err := doQuery(state, q, store)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(val, expected, options...) {
			t.Error(cmp.Diff(val, expected, options...))
		}
	})
}

type storeRequest struct {
	key  string
	path string
}

func splitTestFile(fname string, state *state) (q string, expected map[string]interface{}, store []storeRequest, options cmp.Options, err error) {
	b, err := os.ReadFile("testdata/" + fname)
	if err != nil {
		return "", nil, nil, nil, err
	}

	b = removeComments(b)

	tpl, err := template.New("query").Parse(string(b))
	if err != nil {
		return "", nil, nil, nil, err
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, state.data); err != nil {
		return "", nil, nil, nil, err
	}

	parts := bytes.SplitN(buf.Bytes(), []byte("RETURNS"), 2)
	q = string(parts[0])

	optParts := bytes.SplitN(parts[1], []byte("ENDOPTS"), 2)
	returns := optParts[len(optParts)-1]

	if len(optParts) > 1 {
		os := strings.Split(string(optParts[0]), "OPTION")
		for _, o := range os {
			if strings.TrimSpace(o) == "" {
				continue
			}
			ps := strings.SplitN(o, "=", 2)

			path := strings.TrimSpace(ps[0])
			option := strings.TrimSpace(ps[1])

			switch option {
			case NotNullOption:
				options = append(options, cmp.FilterPath(ignorePath(path), cmp.Comparer(cmpNotNull)))
			case IgnoreOption:
				options = append(options, cmp.FilterPath(ignorePath(path), cmp.Ignore()))
			default:
				return "", nil, nil, nil, fmt.Errorf("unexpected option on path: %v", path)
			}
		}
	}

	srs := bytes.Split(returns, []byte("STORE"))
	if err := json.Unmarshal(srs[0], &expected); err != nil {
		return "", nil, nil, nil, err
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

	return q, expected, store, options, nil
}

func ignorePath(path string) func(p cmp.Path) bool {
	return func(p cmp.Path) bool {
		s := ""
		wide := ""
		for _, pe := range p {
			switch pe := pe.(type) {
			case cmp.MapIndex:
				s += "." + pe.Key().String()
				wide += "." + pe.Key().String()
			case cmp.SliceIndex:
				s += "." + strconv.Itoa(pe.Key())
				wide += ".*"
			}
		}
		return s == "."+path || wide == "."+path
	}
}

func cmpNotNull(a, b interface{}) bool {
	if a == nil || b == nil {
		return false
	}
	return true
}

type state struct {
	data map[string]interface{}
}

func doQuery(state *state, q string, store []storeRequest) (map[string]interface{}, error) {
	b, err := json.Marshal(
		struct {
			OperationName *string                `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
			Query         string                 `json:"query"`
		}{
			nil,
			map[string]interface{}{},
			q,
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

	if e, ok := ret["errors"]; ok {
		fj, _ := json.MarshalIndent(e, "", "  ")
		panic(string(fj))
	}

	for _, s := range store {
		var (
			root = ret
			val  interface{}
		)

		pathParts := strings.Split(s.path, ".")

		for i := 0; i < len(pathParts); i++ {
			if i == len(pathParts)-1 {
				// Last element of pathParts
				val = root[pathParts[i]]
				break
			} else if i < len(pathParts)-2 {
				// check if next value is a list
				intVal, err := strconv.Atoi(pathParts[i+1])
				if err == nil {
					slice := root[pathParts[i]].([]interface{})
					root = slice[intVal].(map[string]interface{})
					// skip one iteration on lists
					i += 1
					continue
				}
			}
			root = root[pathParts[i]].(map[string]interface{})
		}
		state.data[s.key] = val
	}
	return ret, nil
}

func removeComments(b []byte) []byte {
	lines := bytes.Split(b, []byte("\n"))
	ret := [][]byte{}
	for _, l := range lines {
		if !bytes.HasPrefix(l, []byte("//")) {
			ret = append(ret, l)
		}
	}
	return bytes.Join(ret, []byte("\n"))
}
