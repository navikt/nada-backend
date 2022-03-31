package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
)

func main() {
	bucketPath := fmt.Sprintf("%v/%v.csv", "12345", uuid.New())

	fmt.Println(bucketPath)
	/*
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/Users/erikvatt/github.com/navikt/nada-backend/creds.json")
		currentUser := "erik.vattekar@nav.no"

		_ = currentUser

		srcProject := "nada-dev-db2e"
		srcDataset := "test_dataset"
		srcTable := "test_tabell"

		projectID := "nada-dev-db2e"

		bucket := "nada-csv-export-dev"
		object := fmt.Sprintf("%v.csv", srcTable)
		gcsURI := fmt.Sprintf("gs://%v/%v", bucket, object)

		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			log.Fatal(err)
		}
		defer client.Close()

		gcsRef := bigquery.NewGCSReference(gcsURI)
		gcsRef.FieldDelimiter = ";"

		extractor := client.DatasetInProject(srcProject, srcDataset).Table(srcTable).ExtractorTo(gcsRef)
		extractor.Location = "europe-north1"

		job, err := extractor.Run(ctx)
		if err != nil {
			log.Fatal(err)
		}

		statuss, err := job.Status(ctx)
		fmt.Println(statuss.State)
		fmt.Println(err)

		_, err = job.Wait(ctx)
		if err != nil {
			log.Fatal(err)
		}

		client.Location = "europe-north1"
		fmt.Println(job.ID())

		fmt.Println(job.ID())

		fmt.Println(job.LastStatus().Done())
		/*if err := status.Err(); err != nil {
			log.Fatal(err)
		}*/
	/*
		job2, err := client.JobFromID(ctx, job.ID())
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(job2.LastStatus().Done())
	*/
	// createSignedURL(ctx, bucket, object)
}

func createSignedURL(ctx context.Context, bucket, object string) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Signing a URL requires credentials authorized to sign a URL. You can pass
	// these in through SignedURLOptions with one of the following options:
	//    a. a Google service account private key, obtainable from the Google Developers Console
	//    b. a Google Access ID with iam.serviceAccounts.signBlob permissions
	//    c. a SignBytes function implementing custom signing.
	// In this example, none of these options are used, which means the SignedURL
	// function attempts to use the same authentication that was used to instantiate
	// the Storage client. This authentication must include a private key or have
	// iam.serviceAccounts.signBlob permissions.
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(15 * time.Minute),
	}

	u, err := client.Bucket(bucket).SignedURL(object, opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(u)
}
