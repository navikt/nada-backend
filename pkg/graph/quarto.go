package graph

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *mutationResolver) NewQuartoStory(ctx context.Context, file graphql.Upload, input models.NewQuartoStory) (*models.QuartoStory, error) {
	fmt.Println("New Quarto story")
	// Replace with your project ID and GCP bucket name
	bucketName := os.Getenv("GCP_QUARTO_STORAGE_BUCKET_NAME")

	// Create a new GCP storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a new GCP bucket handle
	bucket := client.Bucket(bucketName)

	// Generate a unique filename for the uploaded file
	filename := time.Now().Format("20060102150405") + "-" + file.Filename

	// Create a new GCP object handle
	object := bucket.Object(filename)

	// Create a new GCP object writer
	writer := object.NewWriter(ctx)

	var fileBytes []byte
	_, err = file.File.Read(fileBytes)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(fileBytes)
	// Write the file contents to the GCP object
	if _, err = writer.Write(fileBytes); err != nil {
		return nil, err
	}

	objectAttr, err := object.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	//Create a new File object with the uploaded file's public URL
	return &models.QuartoStory{
		Filename: filename,
		ID:       uuid.New(),
		Name:     file.Filename,
		URL:      objectAttr.MediaLink,
	}, nil
}
