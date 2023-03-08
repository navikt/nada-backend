package graph

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/99designs/gqlgen/graphql"
)

func WriteFileToBucket(ctx context.Context, quartoStoryID string,
	file graphql.Upload,
) error {
	// Replace with your project ID and GCP bucket name
	bucketName := os.Getenv("GCP_QUARTO_STORAGE_BUCKET_NAME")

	// Create a new GCP storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer client.Close()

	// Create a new GCP bucket handle
	bucket := client.Bucket(bucketName)

	// Generate a unique filename for the uploaded file
	filename := quartoStoryID + "/" + file.Filename

	// Create a new GCP object handle
	object := bucket.Object(filename)

	// Create a new GCP object writer
	writer := object.NewWriter(ctx)

	fileBytes, err := ioutil.ReadAll(file.File)
	if err != nil {
		return err
	}

	_, err = writer.Write(fileBytes)
	// Write the file contents to the GCP object
	if _, err = writer.Write(fileBytes); err != nil {
		fmt.Println("failed to write object")
		return err
	}

	if err = writer.Close(); err != nil {
		fmt.Printf("failed to close writer: %v", err)
		return err
	}

	_, err = object.Attrs(ctx)
	if err != nil {
		fmt.Println("failed to fetch object attributes")
		return err
	}

	return nil
}
