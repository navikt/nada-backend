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
	"github.com/navikt/nada-backend/pkg/graph/models"
	"google.golang.org/api/iterator"
)

func WriteFilesToBucket(ctx context.Context, storyID string,
	files []*models.UploadFile,
) error {
	var err error
	for _, file := range files {
		gcsPath := storyID + "/" + file.Path
		err = WriteFileToBucket(ctx, gcsPath, file.File)
		if err != nil {
			log.Fatalf("Error writing story file: " + gcsPath)
			break
		}
	}
	if err != nil {
		ed := deleteStoryFolder(ctx, storyID)
		if ed != nil {
			log.Fatalf("Error delete story folder: " + storyID)
		}
	}

	return err
}

func WriteFileToBucket(ctx context.Context, gcsPath string,
	file graphql.Upload,
) error {
	// Replace with your project ID and GCP bucket name
	bucketName := os.Getenv("GCP_STORY_BUCKET_NAME")

	// Create a new GCP storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer client.Close()

	// Create a new GCP bucket handle
	bucket := client.Bucket(bucketName)

	// Create a new GCP object handle
	object := bucket.Object(gcsPath)

	// Create a new GCP object writer
	writer := object.NewWriter(ctx)

	fileBytes, err := ioutil.ReadAll(file.File)
	if err != nil {
		return err
	}

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

func deleteStoryFolder(ctx context.Context, storyID string) error {
	if len(storyID) == 0 {
		return fmt.Errorf("try to delete files in GCP with invalid story id")
	}
	// Replace with your GCP bucket name.
	bucketName := os.Getenv("GCP_STORY_BUCKET_NAME")

	// Create a client to interact with the GCP Storage API.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Get a handle to the bucket.
	bucket := client.Bucket(bucketName)

	fit := bucket.Objects(ctx, &storage.Query{
		Prefix: storyID + "/",
	})

	var deletedFiles []string
	for {
		f, err := fit.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
			return fmt.Errorf("failed to find objects %v: %v", storyID, err)
		}

		err = bucket.Object(f.Name).Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete %v: %v", f.Name, err)
		}
		deletedFiles = append(deletedFiles, f.Name)
	}

	if len(deletedFiles) == 0 {
		return fmt.Errorf("object not found %v", storyID)
	}

	return nil
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func ptrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
