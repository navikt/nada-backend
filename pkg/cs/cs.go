// Package cs provides a simple API for the Google Cloud Storage service.
package cs

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"strconv"
)

var ErrObjectNotExist = errors.New("object does not exist")
var ErrBucketNotExist = errors.New("bucket does not exist")

type Operations interface {
}

type Client struct {
	client *storage.Client
	bucket string
}

type Object struct {
	Name   string
	Bucket string
	Attrs  Attributes
}

type ObjectWithData struct {
	*Object
	Data []byte
}

type Attributes struct {
	ContentType     string
	ContentEncoding string
	Size            int64
	SizeStr         string
}

type Query struct {
	Prefix string
}

func (c *Client) DeleteObjects(ctx context.Context, q *Query) error {
	var query *storage.Query
	if q != nil {
		query = &storage.Query{
			Prefix: q.Prefix,
		}
	}

	it := c.client.Bucket(c.bucket).Objects(ctx, query)
	for {
		obj, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			if errors.Is(err, storage.ErrBucketNotExist) {
				return ErrBucketNotExist
			}

			return fmt.Errorf("iterating objects: %w", err)
		}

		if err := c.client.Bucket(c.bucket).Object(obj.Name).Delete(ctx); err != nil {
			return fmt.Errorf("deleting object: %w", err)
		}
	}

	return nil
}

func (c *Client) WriteObject(ctx context.Context, name string, data []byte, attrs *Attributes) error {
	obj := c.client.Bucket(c.bucket).Object(name)

	w := obj.NewWriter(ctx)
	defer w.Close()

	if attrs != nil && attrs.ContentType != "" {
		w.ContentType = attrs.ContentType
	}

	if attrs != nil && attrs.ContentEncoding != "" {
		w.ContentEncoding = attrs.ContentEncoding
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing object: %w", err)
	}

	return nil
}

func (c *Client) GetObjects(ctx context.Context, q *Query) ([]*Object, error) {
	var objects []*Object

	var query *storage.Query
	if q != nil {
		query = &storage.Query{
			Prefix: q.Prefix,
		}
	}

	it := c.client.Bucket(c.bucket).Objects(ctx, query)
	for {
		obj, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			if errors.Is(err, storage.ErrBucketNotExist) {
				return nil, ErrBucketNotExist
			}

			return nil, fmt.Errorf("iterating objects: %w", err)
		}

		objects = append(objects, &Object{
			Name:   obj.Name,
			Bucket: obj.Bucket,
			Attrs: Attributes{
				ContentType:     obj.ContentType,
				ContentEncoding: obj.ContentEncoding,
				Size:            obj.Size,
				SizeStr:         strconv.FormatInt(obj.Size, 10),
			},
		})
	}

	return objects, nil

}

func (c *Client) GetObjectWithData(ctx context.Context, name string) (*ObjectWithData, error) {
	obj := c.client.Bucket(c.bucket).Object(name)

	r, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, ErrObjectNotExist
		}

		if errors.Is(err, storage.ErrBucketNotExist) {
			return nil, ErrBucketNotExist
		}

		return nil, fmt.Errorf("creating reader: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading object: %w", err)
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting object attributes: %w", err)
	}

	return &ObjectWithData{
		Object: &Object{
			Name:   obj.ObjectName(),
			Bucket: obj.BucketName(),
			Attrs: Attributes{
				ContentType:     attrs.ContentType,
				ContentEncoding: attrs.ContentEncoding,
				Size:            attrs.Size,
				SizeStr:         strconv.FormatInt(attrs.Size, 10),
			},
		},
		Data: data,
	}, nil
}

func New(ctx context.Context, bucket string) (*Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	return &Client{
		client: client,
		bucket: bucket,
	}, nil
}

func NewFromClient(bucket string, client *storage.Client) *Client {
	return &Client{
		client: client,
		bucket: bucket,
	}
}
