package gcs

import (
	"context"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

type Client struct {
	client     *storage.Client
	bucketName string
	log        *logrus.Entry
}

func New(ctx context.Context, bucketName string, log *logrus.Entry) (*Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:     client,
		bucketName: bucketName,
		log:        log,
	}, nil
}

func (c *Client) GetIndexHtmlPath(ctx context.Context, qID string) (string, error) {
	objs := c.client.Bucket(c.bucketName).Objects(ctx, &storage.Query{Prefix: qID + "/"})
	return c.findIndexPage(qID, objs)
}

func (c *Client) GetObject(ctx context.Context, path string) (*storage.ObjectAttrs, []byte, error) {
	obj := c.client.Bucket(c.bucketName).Object(path)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, []byte{}, err
	}

	datab, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, []byte{}, err
	}

	attr, err := obj.Attrs(ctx)
	if err != nil {
		return nil, []byte{}, err
	}

	return attr, datab, nil
}

func (c *Client) UploadFile(ctx context.Context, name string, file multipart.File) error {
	datab, err := ioutil.ReadAll(file)
	if err != nil {
		c.log.WithError(err).Errorf("reading uploaded file %v", name)
		return err
	}

	writer := c.client.Bucket(c.bucketName).Object(name).NewWriter(ctx)
	_, err = writer.Write(datab)
	if err != nil {
		c.log.WithError(err).Errorf("writing file %v to bucket", name)
		return err
	}

	if err = writer.Close(); err != nil {
		c.log.WithError(err).Errorf("closing writer after writing file %v to bucket", name)
		return err
	}

	return nil
}

func (c *Client) findIndexPage(qID string, objs *storage.ObjectIterator) (string, error) {
	page := ""
	for {
		o, err := objs.Next()
		if err == iterator.Done {
			if page == "" {
				return "", fmt.Errorf("could not find html for id %v", qID)
			}
			return page, nil
		}
		if err != nil {
			c.log.WithError(err).Error("searching for index page in bucket")
			return "", fmt.Errorf("index page not found")
		}

		if strings.HasSuffix(strings.ToLower(o.Name), "/index.html") {
			return o.Name, nil
		} else if strings.HasSuffix(strings.ToLower(o.Name), ".html") {
			page = o.Name
		}
	}
}
