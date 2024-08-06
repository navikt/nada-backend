package cs_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/cs"
	"github.com/navikt/nada-backend/pkg/cs/emulator"
	"github.com/stretchr/testify/assert"
)

func TestClient_DeleteObjects(t *testing.T) {
	testCases := []struct {
		name           string
		bucket         string
		initialObjects []fakestorage.Object
		query          *cs.Query
		expectErr      bool
		expect         any
		count          int
	}{
		{
			name:   "delete objects with prefix",
			bucket: "some-bucket",
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "some/object/file.txt",
					},
				},
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "some/object/file2.txt",
					},
				},
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "not/in/prefix/file2.txt",
					},
				},
			},
			query: &cs.Query{Prefix: "some/object/"},
			expect: []string{
				"not/in/prefix/file2.txt",
			},
			count: 2,
		},
		{
			name:   "delete objects with no prefix",
			bucket: "some-bucket",
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "some/object/file.txt",
					},
				},
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "some/other/path/file2.txt",
					},
				},
			},
			expect: []string{},
			count:  2,
		},
		{
			name:      "no such bucket",
			bucket:    "some-bucket",
			expectErr: true,
			expect:    cs.ErrBucketNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := emulator.New(t, tc.initialObjects)
			defer e.Cleanup()

			client := cs.NewFromClient(tc.bucket, e.Client())

			n, err := client.DeleteObjects(context.Background(), tc.query)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got := e.ListObjectNames(tc.bucket)
				assert.Equal(t, tc.expect, got)
				assert.Equal(t, tc.count, n)
			}
		})
	}
}

func TestClient_WriteObject(t *testing.T) {
	testCases := []struct {
		name           string
		bucket         string
		createBucket   bool
		object         string
		data           []byte
		attrs          *cs.Attributes
		initialObjects []fakestorage.Object
		expectErr      bool
		expect         any
	}{
		{
			name:         "write object",
			bucket:       "some-bucket",
			createBucket: true,
			object:       "some/object/file.txt",
			data:         []byte("inside the file"),
			expect: fakestorage.Object{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName:      "some-bucket",
					Name:            "some/object/file.txt",
					Size:            15,
					ContentType:     "text/plain; charset=utf-8",
					ContentEncoding: "",
				},
				Content: []byte("inside the file"),
			},
		},
		{
			name:         "write object with attrs",
			bucket:       "some-bucket",
			createBucket: true,
			object:       "some/object/file.txt",
			data:         []byte("{}"),
			attrs: &cs.Attributes{
				ContentType:     "application/json",
				ContentEncoding: "utf-8",
			},
			expect: fakestorage.Object{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName:      "some-bucket",
					Name:            "some/object/file.txt",
					Size:            2,
					ContentType:     "application/json",
					ContentEncoding: "utf-8",
				},
				Content: []byte("{}"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := emulator.New(t, tc.initialObjects)
			defer e.Cleanup()

			if tc.createBucket {
				e.CreateBucket(tc.bucket)
			}

			client := cs.NewFromClient(tc.bucket, e.Client())

			err := client.WriteObject(context.Background(), tc.object, io.NopCloser(bytes.NewReader(tc.data)), tc.attrs)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got := e.GetObject(tc.bucket, tc.object)
				diff := cmp.Diff(tc.expect, got,
					cmpopts.IgnoreFields(fakestorage.ObjectAttrs{},
						"Crc32c",
						"Md5Hash",
						"Etag",
						"ACL",
						"Created",
						"Updated",
						"Generation",
					))
				assert.Empty(t, diff)
			}
		})
	}
}

func TestClient_GetObjects(t *testing.T) {
	testCases := []struct {
		name           string
		bucket         string
		query          *cs.Query
		initialObjects []fakestorage.Object
		expect         any
		expectErr      bool
	}{
		{
			name:   "bucket with objects",
			bucket: "some-bucket",
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						BucketName:      "some-bucket",
						Name:            "some/object/file.txt",
					},
					Content: []byte("inside the file"),
				},
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						BucketName:      "some-bucket",
						Name:            "some/object/file2.txt",
					},
				},
			},
			expect: []*cs.Object{
				{
					Name:   "some/object/file.txt",
					Bucket: "some-bucket",
					Attrs: cs.Attributes{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						Size:            15,
						SizeStr:         "15",
					},
				},
				{
					Name:   "some/object/file2.txt",
					Bucket: "some-bucket",
					Attrs: cs.Attributes{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						Size:            0,
						SizeStr:         "0",
					},
				},
			},
		},
		{
			name:   "objects with query",
			bucket: "some-bucket",
			query: &cs.Query{
				Prefix: "some/path2",
			},
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						BucketName:      "some-bucket",
						Name:            "some/path1/file.txt",
					},
					Content: []byte("inside the file"),
				},
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						BucketName:      "some-bucket",
						Name:            "some/path2/file2.txt",
					},
				},
			},
			expect: []*cs.Object{
				{
					Name:   "some/path2/file2.txt",
					Bucket: "some-bucket",
					Attrs: cs.Attributes{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						Size:            0,
						SizeStr:         "0",
					},
				},
			},
		},
		{
			name:      "no such bucket",
			bucket:    "some-bucket",
			expect:    cs.ErrBucketNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := emulator.New(t, tc.initialObjects)
			defer e.Cleanup()

			client := cs.NewFromClient(tc.bucket, e.Client())

			got, err := client.GetObjects(context.Background(), tc.query)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				diff := cmp.Diff(tc.expect, got)
				assert.Empty(t, diff)
			}
		})
	}
}

func TestClient_GetObjectWithAttributes(t *testing.T) {
	testCases := []struct {
		name           string
		bucket         string
		object         string
		initialObjects []fakestorage.Object
		expect         any
		expectErr      bool
	}{
		{
			name:   "object with attributes",
			bucket: "some-bucket",
			object: "some/object/file.txt",
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						BucketName:      "some-bucket",
						Name:            "some/object/file.txt",
					},
					Content: []byte("inside the file"),
				},
			},
			expect: &cs.ObjectWithData{
				Object: &cs.Object{
					Name:   "some/object/file.txt",
					Bucket: "some-bucket",
					Attrs: cs.Attributes{
						ContentType:     "text/plain",
						ContentEncoding: "utf-8",
						Size:            15,
						SizeStr:         "15",
					},
				},
				Data: []byte("inside the file"),
			},
		},
		{
			name:   "no such object",
			bucket: "some-bucket",
			object: "some/object/file.txt",
			initialObjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "some-bucket",
						Name:       "some/object/file2.txt",
					},
				},
			},
			expect:    cs.ErrObjectNotExist,
			expectErr: true,
		},
		{
			name:      "no such bucket",
			bucket:    "some-bucket",
			object:    "some/object/file.txt",
			expect:    cs.ErrBucketNotExist,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := emulator.New(t, tc.initialObjects)
			defer e.Cleanup()

			client := cs.NewFromClient(tc.bucket, e.Client())

			got, err := client.GetObjectWithData(context.Background(), tc.object)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				diff := cmp.Diff(tc.expect, got)
				assert.Empty(t, diff)
			}
		})
	}
}
