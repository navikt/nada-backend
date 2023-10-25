package models

import "github.com/99designs/gqlgen/graphql"

// UploadFile contains path and data of a file
type UploadFile struct {
	// path of the file uploaded
	Path string `json:"path"`
	// file data
	File graphql.Upload `json:"file"`
}
