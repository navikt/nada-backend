package gcs

import (
	"context"
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
