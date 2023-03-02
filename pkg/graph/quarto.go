package graph

import (
	"context"
	"fmt"
	"io/ioutil"
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

	fmt.Println(bucket)

	// Generate a unique filename for the uploaded file
	filename := time.Now().Format("20060102150405") + "-" + file.Filename

	fmt.Println(filename)
	// Create a new GCP object handle
	object := bucket.Object(filename)

	fmt.Println(object)
	// Create a new GCP object writer
	writer := object.NewWriter(ctx)

	fileBytes, err := ioutil.ReadAll(file.File)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(fileBytes))

	_, err = writer.Write(fileBytes)
	// Write the file contents to the GCP object
	if _, err = writer.Write(fileBytes); err != nil {
		fmt.Println("failed to write object")
		return nil, err
	}

	objectAttr, err := object.Attrs(ctx)
	if err != nil {
		fmt.Println("failed to fetch object attributes")
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
