package gcs

import (
	"context"
	"mime/multipart"
)

type ClientMock struct{}

func NewMock() *ClientMock {
	return &ClientMock{}
}

func (m *ClientMock) GetIndexHtmlPath(ctx context.Context, qID string) (string, error) {
	return qID + "/index.html", nil
}

func (m *ClientMock) GetObject(ctx context.Context, path string) ([]byte, error) {
	return []byte("<!DOCTYPE html><html><body><h1>QUARTO</h1></body></html>"), nil
}

func (m *ClientMock) UploadFile(ctx context.Context, name string, file multipart.File) error {
	return nil
}
