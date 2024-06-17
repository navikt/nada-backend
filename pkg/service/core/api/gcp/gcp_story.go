package gcp

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"io"
	"mime/multipart"
	"strings"
)

var _ service.StoryAPI = &storyAPI{}

type storyAPI struct {
	log        *logrus.Entry
	bucketName string
	endpoint   string
}

func (s *storyAPI) GetIndexHtmlPath(ctx context.Context, prefix string) (string, error) {
	client, err := s.newClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	prefix, _ = strings.CutSuffix(prefix, "/")

	_, err = client.Bucket(s.bucketName).Object(prefix + "/index.html").NewReader(ctx)
	if err != nil {
		objs := client.Bucket(s.bucketName).Objects(ctx, &storage.Query{Prefix: prefix + "/"})
		return s.findIndexPage(prefix, objs)
	}

	return prefix + "/index.html", nil
}

func (s *storyAPI) findIndexPage(qID string, objs *storage.ObjectIterator) (string, error) {
	page := ""
	for {
		o, err := objs.Next()
		if errors.Is(err, iterator.Done) {
			if page == "" {
				return "", fmt.Errorf("could not find html for id %v", qID)
			}

			// FIXME: is this correct?
			return page, nil
		}
		if err != nil {
			return "", fmt.Errorf("index page not found")
		}

		if strings.HasSuffix(strings.ToLower(o.Name), "/index.html") {
			return o.Name, nil
		} else if strings.HasSuffix(strings.ToLower(o.Name), ".html") {
			page = o.Name
		}
	}
}

func (s *storyAPI) GetObject(ctx context.Context, path string) (*storage.ObjectAttrs, []byte, error) {
	client, err := s.newClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	obj := client.Bucket(s.bucketName).Object(path)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, err
	}

	datab, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	attr, err := obj.Attrs(ctx)
	if err != nil {
		return nil, nil, err
	}

	return attr, datab, nil
}

func (s *storyAPI) UploadFile(ctx context.Context, name string, file multipart.File) error {
	client, err := s.newClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	datab, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("reading uploaded file %v: %w", name, err)
	}

	writer := client.Bucket(s.bucketName).Object(name).NewWriter(ctx)
	_, err = writer.Write(datab)
	if err != nil {
		return fmt.Errorf("writing file %v to bucket: %w", name, err)
	}

	if err = writer.Close(); err != nil {
		return err
	}

	return nil
}

func (s *storyAPI) DeleteObjectsWithPrefix(ctx context.Context, prefix string) error {
	client, err := s.newClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket(s.bucketName)
	query := &storage.Query{Prefix: prefix}
	it := bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects with prefix: %w", err)
		}

		obj := bucket.Object(attrs.Name)
		if err := obj.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete object: %w", err)
		}
	}

	return nil
}

func (s *storyAPI) WriteFilesToBucket(ctx context.Context, storyID string, files []*service.UploadFile, cleanupOnFailure bool) error {
	var err error

	for _, file := range files {
		gcsPath := storyID + "/" + file.Path
		err = s.WriteFileToBucket(ctx, gcsPath, file.Data)
		if err != nil {
			s.log.WithError(err).Errorf("writing story file: " + gcsPath)
			break
		}
	}
	if err != nil && cleanupOnFailure {
		ed := s.DeleteStoryFolder(ctx, storyID)
		if ed != nil {
			s.log.WithError(ed).Errorf("deleting story folder on cleanup: " + storyID)
		}
	}

	return err
}

func (s *storyAPI) WriteFileToBucket(ctx context.Context, gcsPath string, data []byte) error {
	client, err := s.newClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create a new GCP bucket handle
	bucket := client.Bucket(s.bucketName)

	// Create a new GCP object handle
	object := bucket.Object(gcsPath)

	// Create a new GCP object writer
	writer := object.NewWriter(ctx)

	// Write the file contents to the GCP object
	if _, err = writer.Write(data); err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		return err
	}

	_, err = object.Attrs(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *storyAPI) DeleteStoryFolder(ctx context.Context, storyID string) error {
	if len(storyID) == 0 {
		return fmt.Errorf("try to delete files in GCP with invalid story id")
	}

	client, err := s.newClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Get a handle to the bucket.
	bucket := client.Bucket(s.bucketName)

	fit := bucket.Objects(ctx, &storage.Query{
		Prefix: storyID + "/",
	})

	var deletedFiles []string
	for {
		f, err := fit.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to find objects %v: %v", storyID, err)
		}

		err = bucket.Object(f.Name).Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete %v: %v", f.Name, err)
		}

		deletedFiles = append(deletedFiles, f.Name)
	}

	if len(deletedFiles) == 0 {
		return fmt.Errorf("object not found %v", storyID)
	}

	return nil
}

func (s *storyAPI) newClient(ctx context.Context) (*storage.Client, error) {
	var options []option.ClientOption

	if s.endpoint != "" {
		options = append(options, option.WithEndpoint(s.endpoint))
	}

	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
	}

	return client, nil
}

func NewStoryAPI(endpoint, bucketName string, log *logrus.Entry) *storyAPI {
	return &storyAPI{
		log:        log,
		bucketName: bucketName,
		endpoint:   endpoint,
	}
}
