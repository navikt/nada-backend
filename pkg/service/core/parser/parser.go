package parser

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
)

const (
	HeaderAuthorization          = "Authorization"
	HeaderContentType            = "Content-Type"
	ContentTypeMultipartFormData = "multipart/form-data"
)

var ErrNotExist = errors.New("object not exist")

// Object represents a generic object in a multipart form
type Object struct {
	Name string
	Data []byte
}

// FromJSON unmarshals the data of the object into the provided value
func (o *Object) FromJSON(v interface{}) error {
	err := json.Unmarshal(o.Data, v)
	if err != nil {
		return fmt.Errorf("unmarshalling object %s data: %w", o.Name, err)
	}

	return nil
}

// File represents a file in a multipart form
type File struct {
	Path   string
	Reader io.ReadCloser
}

// DataReadCloser provides a transparent way to read data from a temporary
// file on disk
type DataReadCloser struct {
	file *os.File
}

// Read the data from the temporary file
func (f *DataReadCloser) Read(p []byte) (n int, err error) {
	return bufio.NewReader(f.file).Read(p)
}

// Close and remove the temporary file
func (f *DataReadCloser) Close() error {
	defer func() {
		_ = os.Remove(f.file.Name())
	}()

	err := f.file.Close()
	if err != nil {
		return err
	}

	return nil
}

type MultipartForm struct {
	r       *http.Request
	objects map[string]*Object
	files   []*File
}

func (f *MultipartForm) DeserializedObject(name string, v interface{}) error {
	o, ok := f.objects[name]
	if !ok {
		return ErrNotExist
	}

	return o.FromJSON(v)
}

func (f *MultipartForm) Object(name string) (*Object, error) {
	o, ok := f.objects[name]
	if !ok {
		return nil, ErrNotExist
	}

	return o, nil
}

func (f *MultipartForm) Files() []*File {
	return f.files
}

// Process reads the form body and parses it into objects or files, where
// objectNames is a list of form names that should be treated as objects
// Note: objectNames take precedence over files
func (f *MultipartForm) Process(objectNames []string) error {
	reader, err := f.r.MultipartReader()
	if err != nil {
		return fmt.Errorf("creating multipart reader: %w", err)
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("reading next part: %w", err)
		}

		name := part.FormName()

		// If the part has no name, we skip it
		if len(name) == 0 {
			continue
		}

		// Objects take precedence over files
		if slices.Contains(objectNames, name) {
			data, err := io.ReadAll(part)
			if err != nil {
				return fmt.Errorf("reading part data: %w", err)
			}

			f.objects[name] = &Object{
				Name: name,
				Data: data,
			}

			continue
		}

		// We know we are dealing with a file
		file, err := os.CreateTemp("", "nada-backend-form-data")
		if err != nil {
			return fmt.Errorf("creating temporary file: %w", err)
		}

		// Copy the part data to the temporary file in a streaming fashion
		// to avoid loading the entire file into memory
		_, err = io.Copy(file, part)
		if err != nil {
			return fmt.Errorf("copying part data to file: %w", err)
		}

		// Seek to the start of the file
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("seeking to start of file: %w", err)
		}

		// Add the file to the list of files
		f.files = append(f.files, &File{
			Path: name,
			Reader: &DataReadCloser{
				file: file,
			},
		})
	}

	return nil
}

func MultipartFormFromRequest(r *http.Request) (*MultipartForm, error) {
	contentType := r.Header.Get(HeaderContentType)
	if !strings.Contains(contentType, ContentTypeMultipartFormData) {
		return nil, fmt.Errorf("expected %s to be %s ", HeaderContentType, ContentTypeMultipartFormData)
	}

	return &MultipartForm{
		r:       r,
		objects: map[string]*Object{},
	}, nil
}

func BearerTokenFromRequest(header string, r *http.Request) (string, error) {
	input := r.Header.Get(header)
	if len(input) == 0 {
		return "", fmt.Errorf("missing '%s' header", header)
	}

	parts := strings.SplitN(input, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format, expected 'Bearer <token>'")
	}

	if strings.TrimSpace(strings.ToLower(parts[0])) != "bearer" {
		return "", fmt.Errorf("invalid token format, expected 'Bearer <token>'")
	}

	if strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	return parts[1], nil
}
