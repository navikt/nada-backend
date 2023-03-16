package graph

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/99designs/gqlgen/graphql"
	"google.golang.org/api/iterator"
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

func deleteQuartoStoryFolder(ctx context.Context, quartoStoryID string) error {
	// Replace with your GCP bucket name.
	bucketName := os.Getenv("GCP_QUARTO_STORAGE_BUCKET_NAME")

	// Create a client to interact with the GCP Storage API.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Get a handle to the bucket.
	bucket := client.Bucket(bucketName)

	fit := bucket.Objects(ctx, &storage.Query{
		Prefix: quartoStoryID,
	})

	var deletedFiles []string
	for {
		f, err := fit.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
			return fmt.Errorf("failed to find objects %v: %v", quartoStoryID, err)			
		}

		err = bucket.Object(f.Name).Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete %v: %v", f.Name, err)
		}
		deletedFiles = append(deletedFiles, f.Name)
	}

	if(len(deletedFiles) == 0){
		return fmt.Errorf("object not found %v", quartoStoryID)
	}else{
		log.Printf("Quarto files for %v deleted: %v\n", quartoStoryID, deletedFiles)
	}
	return nil
}
