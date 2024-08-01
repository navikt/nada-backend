package parser_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/navikt/nada-backend/pkg/service/core/parser"
	"github.com/stretchr/testify/assert"
)

func TestMultipartFormFromRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/upload", nil)
	req.Header.Set(parser.HeaderContentType, parser.ContentTypeMultipartFormData)
	_, err := parser.MultipartFormFromRequest(req)
	assert.NoError(t, err)

	req = httptest.NewRequest("POST", "/upload", nil)
	req.Header.Set(parser.HeaderContentType, "application/json")
	_, err = parser.MultipartFormFromRequest(req)
	assert.Error(t, err)
}

func CreateMultipartFormRequest(t *testing.T, files map[string]string, objects map[string]string) *http.Request {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	for path, data := range files {
		part, err := writer.CreateFormFile(path, filepath.Base(path))
		assert.NoError(t, err)
		_, err = part.Write([]byte(data))
		assert.NoError(t, err)
	}

	for name, data := range objects {
		part, err := writer.CreateFormField(name)
		assert.NoError(t, err)
		_, err = part.Write([]byte(data))
		assert.NoError(t, err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestMultipartForm_Files(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
	}{
		{
			name:  "no files",
			files: map[string]string{},
		},
		{
			name: "one file",
			files: map[string]string{
				"index.html": "<html>Index content</html>",
			},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"index.html":                      "<html>Index content</html>",
				"subpages/test.html":              "<html>Test content</html>",
				"subpages/test2.html":             "<html>Test2 content</html>",
				"subpages/subsubpages/test3.html": "<html>Test3 content</html>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := CreateMultipartFormRequest(t, tc.files, nil)
			p, err := parser.MultipartFormFromRequest(req)
			assert.NoError(t, err)

			err = p.Process(nil)
			assert.NoError(t, err)

			processedFiles := p.Files()
			assert.Len(t, processedFiles, len(tc.files))

			for _, got := range processedFiles {
				d, err := io.ReadAll(got.Reader)
				assert.NoError(t, err)

				err = got.Reader.Close()
				assert.NoError(t, err)

				assert.Equal(t, tc.files[got.Path], string(d))
			}
		})
	}
}

func TestMultipartForm_DeserializedObject(t *testing.T) {
	testCases := []struct {
		name      string
		names     []string
		objects   map[string]string
		expectErr bool
		expect    any
	}{
		{
			name:      "no objects",
			names:     []string{"not-exist"},
			objects:   map[string]string{},
			expect:    parser.ErrNotExist,
			expectErr: true,
		},
		{
			name:  "one object",
			names: []string{"object1"},
			objects: map[string]string{
				"object1": `{"key": "value"}`,
			},
			expect: map[string]map[string]string{
				"object1": {"key": "value"},
			},
		},
		{
			name:  "multiple objects, two requested",
			names: []string{"object1", "object2"},
			objects: map[string]string{
				"object1": `{"key": "value"}`,
				"object2": `{"key2": "value2"}`,
				"object3": `{"key3": "value3"}`,
			},
			expect: map[string]map[string]string{
				"object1": {"key": "value"},
				"object2": {"key2": "value2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := CreateMultipartFormRequest(t, nil, tc.objects)
			p, err := parser.MultipartFormFromRequest(req)
			assert.NoError(t, err)

			err = p.Process(tc.names)
			assert.NoError(t, err)

			for _, name := range tc.names {
				var v map[string]string
				err := p.DeserializedObject(name, &v)
				if tc.expectErr {
					assert.Error(t, err)
					assert.Equal(t, tc.expect, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expect.(map[string]map[string]string)[name], v)
				}
			}
		})
	}
}

func requestWithHeaders(headers map[string]string) *http.Request {
	req := httptest.NewRequest("GET", "/test", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}

func TestBearerTokenFromRequest(t *testing.T) {
	testCases := []struct {
		name      string
		header    string
		headers   map[string]string
		expect    string
		expectErr bool
	}{
		{
			name:   "bearer token",
			header: parser.HeaderAuthorization,
			headers: map[string]string{
				parser.HeaderAuthorization: "Bearer mysecrettoken",
			},
			expect: "mysecrettoken",
		},
		{
			name:      "no header",
			header:    "",
			headers:   map[string]string{},
			expect:    "missing '' header",
			expectErr: true,
		},
		{
			name:   "wrong header",
			header: "X-Auth",
			headers: map[string]string{
				parser.HeaderAuthorization: "Bearer mysenretoken",
			},
			expect:    "missing 'X-Auth' header",
			expectErr: true,
		},
		{
			name:   "invalid token format",
			header: parser.HeaderAuthorization,
			headers: map[string]string{
				parser.HeaderAuthorization: "Bearer",
			},
			expect:    "invalid token format, expected 'Bearer <token>'",
			expectErr: true,
		},
		{
			name:   "empty token",
			header: parser.HeaderAuthorization,
			headers: map[string]string{
				parser.HeaderAuthorization: "Bearer ",
			},
			expect:    "empty bearer token",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := requestWithHeaders(tc.headers)
			token, err := parser.BearerTokenFromRequest(tc.header, req)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tc.expect, err.Error())
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expect, token)
			}
		})
	}
}
